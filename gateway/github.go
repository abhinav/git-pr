package gateway

import (
	"context"

	"github.com/google/go-github/github"
)

// GitHub is a gateway that provides access to GitHub operations on a specific
// repository.
type GitHub interface {
	// Checks if the given pull request branch is owned by the same
	// repository.
	IsOwned(ctx context.Context, br *github.PullRequestBranch) bool

	// List pull requests on this repository with the given head. If owner is
	// empty, the current repository should be used.
	ListPullRequestsByHead(ctx context.Context, owner, branch string) ([]*github.PullRequest, error)

	// List pull requests on this repository with the given merge base.
	ListPullRequestsByBase(ctx context.Context, branch string) ([]*github.PullRequest, error)

	// Retrieve the raw patch for the given pull request.
	GetPullRequestPatch(ctx context.Context, number int) (string, error)

	// Change the merge base for the given pull request.
	SetPullRequestBase(ctx context.Context, number int, base string) error

	// Merges the given pull request.
	SquashPullRequest(context.Context, *github.PullRequest) error

	// Delete the given branch.
	DeleteBranch(ctx context.Context, name string) error
}
