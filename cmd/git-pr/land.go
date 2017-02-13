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

	prs, _, err := cfg.GitHub().PullRequests.List(repo.Owner, repo.Name,
		&github.PullRequestListOptions{Head: repo.Owner + ":" + branch})
	if err != nil {
		return err
	}

	var pull *github.PullRequest
	switch len(prs) {
	case 0:
		return fmt.Errorf("Could not find PRs with head %q", branch)
	case 1:
		pull = prs[0]
	default:
		return errTooManyPRsWithHead{Head: branch, Pulls: prs}
	}

	log.Println("Landing", pr.URL(pull))

	lander := &pr.InteractiveLander{
		Editor: editor,
		Pulls:  cfg.GitHub().PullRequests,
	}
	if err := lander.Land(pull); err != nil {
		return fmt.Errorf("Could not land %v: %v", pr.URL(pull), err)
	}

	prs, _, err = cfg.GitHub().PullRequests.List(repo.Owner, repo.Name,
		&github.PullRequestListOptions{Base: branch})
	if err == nil && len(prs) == 0 {
		log.Printf("Deleting remote branch %q\n", branch)
		// TODO: if len(prs) > 0, maybe we should rebase those PRs on master
		if _, err := cfg.GitHub().Git.DeleteRef(repo.Owner, repo.Name, "heads/"+branch); err != nil {
			return fmt.Errorf("could not delete remote branch %q: %v", branch, err)
		}
	}

	if deleteLocalBranch {
		if err := git.Run("checkout", *pull.Base.Ref); err != nil {
			return fmt.Errorf("could not switch to loca branch %v: %v", *pull.Base.Ref, err)
		}

		if err := git.Run("pull", "origin", *pull.Base.Ref); err != nil {
			return fmt.Errorf("could not fetch origin: %v", err)
		}

		if err := git.Run("branch", "-D", branch); err != nil {
			return fmt.Errorf("could not delete local branch %v: %v", branch, err)
		}
	}

	return nil
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
