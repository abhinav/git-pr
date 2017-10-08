package service

import (
	"context"

	"github.com/abhinav/git-pr/editor"

	"github.com/google/go-github/github"
)

// LandRequest is a request to land the given pull request.
type LandRequest struct {
	// PullRqeuest to land
	PullRequest *github.PullRequest

	// Name of the local branch that points to this PR or an empty string if a
	// local branch for this PR is not known.
	LocalBranch string

	// Editor to use for editing the commit message.
	Editor editor.Editor
}

// LandResponse is the response of a land request.
type LandResponse struct {
	BranchesNotUpdated []string
}

// RebaseRequest is a request to rebase the given list of pull requests and
// their dependencies onto the given base branch.
//
// If the base branch for the given PRs on GitHub is not the same as Base,
// this will be updated too.
type RebaseRequest struct {
	PullRequests []*github.PullRequest
	Base         string

	// If non-empy, only pull requests by the given user will be rebased.
	Author string
}

// RebaseResponse is the response of the Rebase operation.
type RebaseResponse struct {
	// Local branches that were not updated because their heads did not match
	// the remotes.
	BranchesNotUpdated []string
}

// PR is the service that provides pull request related operations.
type PR interface {
	// Lands a pull request
	Land(context.Context, *LandRequest) (*LandResponse, error)

	// Rebases a pull request.
	Rebase(context.Context, *RebaseRequest) (*RebaseResponse, error)
}
