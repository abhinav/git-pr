package main

import (
	"fmt"
	"log"

	"github.com/abhinav/git-fu/cli"
	"github.com/abhinav/git-fu/editor"
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

	editor, err := editor.Pick(l.Editor)
	if err != nil {
		return err
	}

	req := pr.LandRequest{Editor: editor}
	svc := pr.Service{GitHub: cfg.GitHub(), Git: cfg.Git()}

	// TODO: accept other inputs for the PR to land
	branch := l.Args.Branch
	if branch == "" {
		out, err := cfg.Git().CurrentBranch()
		if err != nil {
			return err
		}
		branch = out
		req.LocalBranch = out
	}

	prs, err := cfg.GitHub().ListPullRequestsByHead("", branch)
	if err != nil {
		return err
	}
	switch len(prs) {
	case 0:
		return fmt.Errorf("Could not find PRs with head %q", branch)
	case 1:
		req.PullRequest = prs[0]
	default:
		return errTooManyPRsWithHead{Head: branch, Pulls: prs}
	}

	log.Println("Landing", *req.PullRequest.HTMLURL)
	res, err := svc.Land(&req)
	if err != nil {
		return fmt.Errorf("failed to land %v: %v", *req.PullRequest.HTMLURL, err)
	}

	if len(res.BranchesNotUpdated) > 0 {
		log.Println("The following local branches were not updated because " +
			"they did not match the corresponding remotes")
		for _, br := range res.BranchesNotUpdated {
			log.Println(" -", br)
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
		msg += fmt.Sprintf("\n -  %v", *pull.HTMLURL)
	}
	msg += fmt.Sprintf("\nPlease provide the PR number instead.")
	return msg
}
