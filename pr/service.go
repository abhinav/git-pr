package pr

import (
	"fmt"

	"github.com/abhinav/git-fu/editor"
	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/internal"

	"github.com/google/go-github/github"
)

// LandRequest is a request to land the given pull request.
type LandRequest struct {
	// PullRqeuest to land
	PullRequest *github.PullRequest

	// Name of the local branch that points to this PR or an empty string if a
	// local branch for this PR is not known.
	LocalBranch string
}

// Service TODO
type Service struct {
	GitHub gateway.GitHub
	Editor editor.Editor
	Git    gateway.Git
}

// Land the given pull request.
func (s *Service) Land(req *LandRequest) error {
	pr := req.PullRequest
	if err := UpdateMessage(s.Editor, pr); err != nil {
		return err
	}

	// If the base branch doesn't exist locally, check it out. If it exists,
	// it's okay for it to be out of sync with the remote.
	base := *pr.Base.Ref
	if !s.Git.DoesBranchExist(base) {
		if err := s.Git.CreateBranch(base, *pr.Base.Ref); err != nil {
			return err
		}
	}

	// If the branch is checked out locally, make sure it's in sync with
	// remote.
	if req.LocalBranch != "" {
		hash, err := s.Git.SHA1(req.LocalBranch)
		if err != nil {
			return err
		}

		if hash != *pr.Head.SHA {
			return fmt.Errorf(
				"SHA1 of local branch %v of pull request %v does not match GitHub. "+
					"Make sure that your local checkout of %v is in sync.",
				req.LocalBranch, *pr.HTMLURL, req.LocalBranch)
		}
	}

	if err := s.GitHub.SquashPullRequest(pr); err != nil {
		return err
	}

	if err := s.Git.Checkout(base); err != nil {
		return err
	}

	// TODO: Remove hard coded remote name
	if err := s.Git.Pull("origin", base); err != nil {
		return err
	}

	if req.LocalBranch != "" {
		if err := s.Git.DeleteBranch(req.LocalBranch); err != nil {
			return err
		}
	}

	// Nothing else to do if we don't own this pull request.
	if !s.GitHub.IsOwned(pr.Head) {
		return nil
	}

	dependents, err := s.GitHub.ListPullRequestsByBase(*pr.Head.Ref)
	if err != nil {
		return err
	}

	// No dependents. Delete the remote branch and the local tracking branch
	// for it.
	if len(dependents) == 0 {
		if err := s.GitHub.DeleteBranch(*pr.Head.Ref); err != nil {
			return err
		}

		if req.LocalBranch != "" {
			// TODO: Remove hard coded remote name
			if err := s.Git.DeleteRemoteTrackingBranch("origin", req.LocalBranch); err != nil {
				return err
			}
		}
	}

	return s.rebaseAll(base, dependents)
}

// Rebase all of the given pull requests onto the given base branch.
func (s *Service) rebaseAll(base string, prs []*github.PullRequest) error {
	var errors []error
	for _, pr := range prs {
		// We don't own this branch so we can't rebase it.
		if !s.GitHub.IsOwned(pr.Head) {
			// TODO record somewhere which PRs got skipped?
			continue
		}

		if err := s.rebaseOnto(base, pr); err != nil {
			errors = append(errors, err)
		}
	}

	return internal.MultiError(errors...)
}

// Rebase a specific PR and its dependents
func (s *Service) rebaseOnto(base string, pr *github.PullRequest) (err error) {
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

	if err := s.GitHub.SetPullRequestBase(*pr.Number, base); err != nil {
		return err
	}

	// If this PR had any dependents, rebase them onto its new head.
	dependents, err := s.GitHub.ListPullRequestsByBase(*pr.Head.Ref)
	if err != nil {
		return err
	}

	if len(dependents) > 0 {
		return s.rebaseAll(*pr.Head.Ref, dependents)
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
