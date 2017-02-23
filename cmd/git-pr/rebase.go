package main

import (
	"fmt"
	"log"

	"github.com/abhinav/git-fu/cli"
	"github.com/abhinav/git-fu/pr"

	"github.com/jessevdk/go-flags"
)

type rebaseCmd struct {
	Base string `long:"onto" value-name:"BASE" description:"Name of the base branch. If unspecified, only the dependents of the current branch will be rebased onto it."`
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

	if len(prs) == 0 {
		return fmt.Errorf("Could not find PRs with head %q", branch)
	}

	var req pr.RebaseRequest
	if r.Base == "" {
		if len(prs) > 1 {
			return errTooManyPRsWithHead{Head: branch, Pulls: prs}
		}

		head := *prs[0].Head.Ref
		dependents, err := cfg.GitHub().ListPullRequestsByBase(head)
		if err != nil {
			return err
		}
		req = pr.RebaseRequest{PullRequests: dependents, Base: head}
	} else {
		req = pr.RebaseRequest{PullRequests: prs, Base: r.Base}
	}

	log.Println("Rebasing:")
	for _, pr := range req.PullRequests {
		log.Printf(" - %v", *pr.HTMLURL)
	}

	res, err := svc.Rebase(&req)
	if err != nil {
		return err
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
