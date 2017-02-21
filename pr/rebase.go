package pr

import (
	"fmt"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/internal"
	"github.com/google/go-github/github"
)

// RebaseRequest is a request to rebase the given list of pull requests and
// their dependencies onto the given base branch.
type RebaseRequest struct {
	PullRequests []*github.PullRequest
	Base         string
}

// Rebase a pull request and its dependencies.
func (s *Service) Rebase(req *RebaseRequest) error {
	// TODO: Check out base branch (as it stands right now) only once
	var errors []error
	for _, pr := range req.PullRequests {
		// We don't own this branch so we can't rebase it.
		if !s.GitHub.IsOwned(pr.Head) {
			// TODO record somewhere which PRs got skipped?
			continue
		}

		if err := rebase(s, req.Base, pr); err != nil {
			err = fmt.Errorf("failed to rebase %v onto %q: %v", *pr.HTMLURL, req.Base, err)
			errors = append(errors, err)
		}
	}

	return internal.MultiError(errors...)
}

func rebase(s *Service, base string, pr *github.PullRequest) (err error) {
	patch, err := s.GitHub.GetPullRequestPatch(*pr.Number)
	if err != nil {
		return err
	}

	// Create a temporary branch off of the base to apply the patch onto.
	tempBranch := temporaryNameFor(s.Git, pr)
	// TODO: We need to create the temporary base branch only once for the
	// same merge base. This would make the operation faster for wider trees.

	fetch := gateway.FetchRequest{
		Remote:    "origin",
		RemoteRef: base,
		LocalRef:  tempBranch,
	}
	if err := s.Git.Fetch(&fetch); err != nil {
		return err
	}
	defer func() {
		err = internal.MultiError(err, s.Git.DeleteBranch(tempBranch))
	}()

	if err := s.Git.Checkout(tempBranch); err != nil {
		return err
	}
	defer func() {
		err = internal.MultiError(err, s.Git.Checkout(base))
	}()

	if err := s.Git.ApplyPatches(patch); err != nil {
		return err
	}

	// If we applied everything successfully, force push the change and update
	// the PR base.
	push := gateway.PushRequest{
		Remote:    "origin",
		LocalRef:  tempBranch,
		RemoteRef: *pr.Head.Ref,
		Force:     true,
	}
	if err := s.Git.Push(&push); err != nil {
		return err
	}

	if *pr.Base.Ref != base {
		if err := s.GitHub.SetPullRequestBase(*pr.Number, base); err != nil {
			return err
		}
	}

	// TODO: If this worked out, we should probably reset the local branch for
	// this PR (if any) to the new head. Maybe by verifying a SHA before
	// rebasing.

	// If this PR had any dependents, rebase them onto its new head.
	dependents, err := s.GitHub.ListPullRequestsByBase(*pr.Head.Ref)
	if err != nil {
		return err
	}

	if len(dependents) > 0 {
		return s.Rebase(&RebaseRequest{
			PullRequests: dependents,
			Base:         *pr.Head.Ref,
		})
	}

	return nil
}

func temporaryNameFor(git gateway.Git, pr *github.PullRequest) string {
	base := fmt.Sprintf("pr-%v-rebase", *pr.Number)
	name := base
	for i := 1; git.DoesBranchExist(name); i++ {
		name = fmt.Sprintf("%v-%v", base, i)
	}
	return name
}
