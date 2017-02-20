package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/abhinav/git-fu/cli"
	"github.com/abhinav/git-fu/editor"
	"github.com/abhinav/git-fu/git"
	"github.com/abhinav/git-fu/pr"

	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
)

type landCmd struct {
	Rebase bool   `long:"rebase" description:"Rebase dependent PRs onto the new merge base."`
	Editor string `long:"editor" env:"EDITOR" default:"vi" value-name:"EDITOR" description:"Editor to use for interactively editing commit messages."`
	Args   struct {
		Branch string `positional-arg-name:"BRANCH" description:"Name of the branch to land. Defaults to the branch in the current directory."`
	} `positional-args:"yes"`

	getConfig cli.ConfigBuilder
}

func newLandCommand(cbuild cli.ConfigBuilder) flags.Commander {
	return &landCmd{getConfig: cbuild}
}

func (l *landCmd) Execute(args []string) error {
	cfg, err := l.getConfig()
	if err != nil {
		return err
	}

	repo := cfg.Repo()

	branch := l.Args.Branch
	deleteLocalBranch := false
	if branch == "" {
		out, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return fmt.Errorf("Could not determine current branch: %v", err)
		}
		branch = strings.TrimSpace(out)
		deleteLocalBranch = true
	}

	editor, err := editor.Pick(l.Editor)
	if err != nil {
		return fmt.Errorf("Could not determine editor: %v", err)
	}

	var pull *github.PullRequest
	{
		prs, _, err := cfg.GitHub().PullRequests.List(repo.Owner, repo.Name,
			&github.PullRequestListOptions{Head: repo.Owner + ":" + branch})
		if err != nil {
			return err
		}

		switch len(prs) {
		case 0:
			return fmt.Errorf("Could not find PRs with head %q", branch)
		case 1:
			pull = prs[0]
		default:
			return errTooManyPRsWithHead{Head: branch, Pulls: prs}
		}
	}

	log.Println("Landing", pr.URL(pull))
	lander := &pr.InteractiveLander{
		Editor: editor,
		Pulls:  cfg.GitHub().PullRequests,
	}
	if err := lander.Land(pull); err != nil {
		return fmt.Errorf("Could not land %v: %v", pr.URL(pull), err)
	}

	if err := git.Run("checkout", *pull.Base.Ref); err != nil {
		return fmt.Errorf("could not switch to loca branch %v: %v", *pull.Base.Ref, err)
	}

	if err := git.Run("pull", "origin", *pull.Base.Ref); err != nil {
		return fmt.Errorf("could not fetch origin: %v", err)
	}

	hasDependents, err := l.rebaseDependents(cfg, *pull.Base.Ref, pull)
	if err == nil && !hasDependents {
		log.Printf("Deleting remote branch %q\n", branch)
		if _, err := cfg.GitHub().Git.DeleteRef(repo.Owner, repo.Name, "heads/"+branch); err != nil {
			return fmt.Errorf("could not delete remote branch %q: %v", branch, err)
		}

		if err := git.Run("branch", "-dr", "origin/"+branch); err != nil {
			return fmt.Errorf("failed to delete remote tracking branch %q: %v", branch, err)
		}
	}

	// We delete the local branch only if it was automatically determined
	// to be the current branch.
	if deleteLocalBranch {
		if err := git.Run("branch", "-D", branch); err != nil {
			return fmt.Errorf("could not delete local branch %v: %v", branch, err)
		}
	}

	return nil
}

// Rebases dependent branches of this PR onto the given base branch.
func (l *landCmd) rebaseDependents(cfg cli.Config, branch string, pull *github.PullRequest) (hasDependents bool, err error) {
	repo := pull.Base.Repo
	prs, _, err := cfg.GitHub().PullRequests.List(*repo.Owner.Login, *repo.Name,
		&github.PullRequestListOptions{Base: *pull.Head.Ref})
	if err != nil {
		return false, fmt.Errorf("could not determine dependent PRs for %v: %v", pr.URL(pull), err)
	}

	hasDependents = len(prs) > 0
	if !l.Rebase {
		return hasDependents, nil
	}

	var errors []error
	for _, p := range prs {
		err := l.rebaseOnto(cfg, *pull.Base.Ref, p)
		if err == nil {
			continue
		}

		err = fmt.Errorf("failed to rebase %v onto %v: %v", pr.URL(p), *pull.Base.Ref, err)
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		err = errMulti(errors)
	}

	return hasDependents, err
}

// Rebases changes in the given pull request onto the given branch. This also
// updates the base branch for the PR.
func (l *landCmd) rebaseOnto(cfg cli.Config, branch string, pull *github.PullRequest) error {
	repo := pull.Base.Repo
	patch, _, err := cfg.GitHub().PullRequests.GetRaw(
		*repo.Owner.Login,
		*repo.Name,
		*pull.Number,
		github.RawOptions{Type: github.Patch},
	)
	if err != nil {
		return fmt.Errorf("failed to get patch for %v", pr.URL(pull))
	}

	tmpBranch := rebaseBranchName(pull)
	if err := git.Run("checkout", "-b", tmpBranch, branch); err != nil {
		return fmt.Errorf("failed to create temporary branch %v: %v", branch, err)
	}
	defer func() {
		err := git.Run("checkout", branch)
		if err == nil {
			err = git.Run("branch", "-D", tmpBranch)
		}
		if err != nil {
			log.Printf("failed to delete temporary branch %v: %v", tmpBranch, err)
		}
	}()

	if err := git.ApplyPatches(strings.NewReader(patch)); err != nil {
		return fmt.Errorf("failed to apply patch: %v", err)
	}

	if err := git.Run("push", "-f", "origin", fmt.Sprintf("%v:%v", tmpBranch, *pull.Head.Ref)); err != nil {
		return fmt.Errorf("failed to push changes to %v: %v", *pull.Head.Ref, err)
	}

	_, _, err = cfg.GitHub().PullRequests.Edit(
		*repo.Owner.Login, *repo.Name, *pull.Number,
		&github.PullRequest{Base: &github.PullRequestBranch{Ref: &branch}})
	if err != nil {
		return fmt.Errorf("failed to change merge base to %v: %v", branch, err)
	}

	if _, err := l.rebaseDependents(cfg, *pull.Head.Ref, pull); err != nil {
		return fmt.Errorf("failed to rebase dependents of %v: %v", pr.URL(pull), err)
	}
	return nil
}

func rebaseBranchName(pull *github.PullRequest) string {
	base := *pull.Head.Ref + "-rebase"
	name := base
	for i := 1; doesBranchExist(name); i++ {
		name = fmt.Sprintf("%v-%d", base, i)
	}
	return name
}

func doesBranchExist(name string) bool {
	_, err := git.Output("show-ref", "refs/heads/"+name)
	return err == nil
}

type errTooManyPRsWithHead struct {
	Head  string
	Pulls []*github.PullRequest
}

func (e errTooManyPRsWithHead) Error() string {
	msg := fmt.Sprintf("Too many PRs found with head %q:", e.Head)
	for _, pull := range e.Pulls {
		msg += fmt.Sprintf("\n -  %v", pr.URL(pull))
	}
	msg += fmt.Sprintf("\nPlease provide the PR number instead.")
	return msg
}

type errMulti []error

func (errs errMulti) Error() string {
	msg := "The following errors occurred:"
	for _, err := range errs {
		msg += "\n -  " + indentTail(4, err.Error())
	}
	return msg
}

// indentTail prepends the given number of spaces to all lines following the
// first line of the given string.
func indentTail(spaces int, s string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines[1:] {
		lines[i+1] = prefix + line
	}
	return strings.Join(lines, "\n")
}
