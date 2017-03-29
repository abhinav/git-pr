package gateway

import (
	"context"

	"github.com/google/go-github/github"
)

// PullRequestReviewState indicates whether a PR has been accepted or not.
type PullRequestReviewState string

const (
	// PullRequestApproved indicates that a pull request was accepted.
	PullRequestApproved PullRequestReviewState = "APPROVED"

	// PullRequestChangesRequested indicates that changes were requested for a
	// pull request.
	PullRequestChangesRequested PullRequestReviewState = "CHANGES_REQUESTED"
)

// PullRequestReview is a review of a pull request.
type PullRequestReview struct {
	// User who did the review.
	User string

	// Whether they approved or requested changes.
	Status PullRequestReviewState
}

// GitHub is a gateway that provides access to GitHub operations on a specific
// repository.
type GitHub interface {
	// Checks if the given pull request branch is owned by the same
	// repository.
	IsOwned(ctx context.Context, br *github.PullRequestBranch) bool

	// Lists reviews for a pull request.
	ListPullRequestReviews(ctx context.Context, number int) ([]*PullRequestReview, error)

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
