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

	// PullRequestCommented indicates that someone commented on a pull
	// requestest without an explicit approval or changes-requested.
	PullRequestCommented PullRequestReviewState = "COMMENTED"

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

// BuildState indicates whether a build succeeded, failed or is pending.
type BuildState string

// All possible BuildStates.
const (
	BuildError   BuildState = "error"
	BuildFailure BuildState = "failure"
	BuildPending BuildState = "pending"
	BuildSuccess BuildState = "success"
)

//BuildContextStatus is the status of a specific build context.
type BuildContextStatus struct {
	Name    string
	Message string
	State   BuildState
}

// BuildStatus indicates the build status of a ref.
type BuildStatus struct {
	State    BuildState
	Statuses []*BuildContextStatus
}

// GitHub is a gateway that provides access to GitHub operations on a specific
// repository.
type GitHub interface {
	// Checks if the given pull request branch is owned by the same
	// repository.
	IsOwned(ctx context.Context, br *github.PullRequestBranch) bool

	// Lists reviews for a pull request.
	ListPullRequestReviews(ctx context.Context, number int) ([]*PullRequestReview, error)

	// Get the build status of a specific ref.
	GetBuildStatus(ctx context.Context, ref string) (*BuildStatus, error)

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
	// TODO: SquashPullRequest should accept an explicit SquashRequest

	// Delete the given branch.
	DeleteBranch(ctx context.Context, name string) error
}
