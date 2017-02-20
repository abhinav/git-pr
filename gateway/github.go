package gateway

import "github.com/google/go-github/github"

// GitHub is a gateway that provides access to GitHub operations on a specific
// repository.
type GitHub interface {
	// Checks if the given pull request branch is owned by the same
	// repository.
	IsOwned(br *github.PullRequestBranch) bool

	// List pull requests on this repository with the given head. If owner is
	// empty, the current repository should be used.
	ListPullRequestsByHead(owner, branch string) ([]*github.PullRequest, error)

	// List pull requests on this repository with the given merge base.
	ListPullRequestsByBase(branch string) ([]*github.PullRequest, error)

	// Retrieve the raw patch for the given pull request.
	GetPullRequestPatch(number int) (string, error)

	// Change the merge base for the given pull request.
	SetPullRequestBase(number int, base string) error

	// Merges the given pull request.
	SquashPullRequest(*github.PullRequest) error

	// Delete the given branch.
	DeleteBranch(name string) error
}
