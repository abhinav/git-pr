package main

import (
	"fmt"
	"log"

	"github.com/abhinav/git-fu/cli"
	"github.com/abhinav/git-fu/pr"
	"github.com/jessevdk/go-flags"
)

type rebaseCmd struct {
	Base string `long:"onto" required:"yes" value-name:"BASE" description:"Name of the base branch"`
	Args struct {
		Branch string `positional-arg-name:"BRANCH" description:"Name of the branch to rebase. Defaults to the branch in the current directory."`
	} `positional-args:"yes"`

	getConfig cli.ConfigBuilder
}

func newRebaseCommand(cbuild cli.ConfigBuilder) flags.Commander {
	return &rebaseCmd{getConfig: cbuild}
}

func (r *rebaseCmd) Execute(args []string) error {
	cfg, err := r.getConfig()
	if err != nil {
		return err
	}

	svc := pr.Service{GitHub: cfg.GitHub(), Git: cfg.Git()}

	// TODO: accept other inputs for the PR to land
	branch := r.Args.Branch
	if branch == "" {
		out, err := cfg.Git().CurrentBranch()
		if err != nil {
			return err
		}
		branch = out
	}

	prs, err := cfg.GitHub().ListPullRequestsByHead("", branch)
	if err != nil {
		return err
	}

	switch len(prs) {
	case 0:
		return fmt.Errorf("Could not find PRs with head %q", branch)
	case 1:
		log.Println("Rebasing", *prs[0].HTMLURL)
	default:
		log.Println("Rebasing:")
		for _, pr := range prs {
			log.Printf(" - %v", *pr.HTMLURL)
		}
	}

	return svc.Rebase(&pr.RebaseRequest{PullRequests: prs, Base: r.Base})
}
