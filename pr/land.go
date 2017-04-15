package pr

import (
	"context"
	"fmt"

	"github.com/abhinav/git-fu/service"
)

// Land the given pull request.
func (s *Service) Land(ctx context.Context, req *service.LandRequest) (*service.LandResponse, error) {
	pr := req.PullRequest
	if !req.NoCheck {
		if err := s.LandCheck.Check(ctx, req.PullRequest); err != nil {
			return nil, fmt.Errorf("cannot land %v: %+v", pr.GetHTMLURL(), err)
		}
	}

	if err := UpdateMessage(req.Editor, pr); err != nil {
		return nil, err
	}

	// If the base branch doesn't exist locally, check it out. If it exists,
	// it's okay for it to be out of sync with the remote.
	base := *pr.Base.Ref
	if !s.Git.DoesBranchExist(base) {
		if err := s.Git.CreateBranch(base, *pr.Base.Ref); err != nil {
			return nil, err
		}
	}

	// If the branch is checked out locally, make sure it's in sync with
	// remote.
	if req.LocalBranch != "" {
		hash, err := s.Git.SHA1(req.LocalBranch)
		if err != nil {
			return nil, err
		}

		if hash != *pr.Head.SHA {
			return nil, fmt.Errorf(
				"SHA1 of local branch %v of pull request %v does not match GitHub. "+
					"Make sure that your local checkout of %v is in sync.",
				req.LocalBranch, *pr.HTMLURL, req.LocalBranch)
		}
	}

	if err := s.GitHub.SquashPullRequest(ctx, pr); err != nil {
		return nil, err
	}

	if err := s.Git.Checkout(base); err != nil {
		return nil, err
	}

	// TODO: Remove hard coded remote name
	if err := s.Git.Pull("origin", base); err != nil {
		return nil, err
	}

	if req.LocalBranch != "" {
		if err := s.Git.DeleteBranch(req.LocalBranch); err != nil {
			return nil, err
		}
	}

	// Nothing else to do if we don't own this pull request.
	if !s.GitHub.IsOwned(ctx, pr.Head) {
		return nil, nil
	}

	dependents, err := s.GitHub.ListPullRequestsByBase(ctx, *pr.Head.Ref)
	if err != nil {
		return nil, err
	}

	var res service.LandResponse
	if len(dependents) > 0 {
		rebaseRes, err := s.Rebase(ctx, &service.RebaseRequest{PullRequests: dependents, Base: base})
		if err != nil {
			return nil, fmt.Errorf("failed to rebase dependents of %v: %v", *pr.HTMLURL, err)
		}
		res.BranchesNotUpdated = rebaseRes.BranchesNotUpdated
	}

	// TODO: What happens on branch deletion if we had dependents but none
	// were owned by us?
	if err := s.GitHub.DeleteBranch(ctx, *pr.Head.Ref); err != nil {
		return nil, err
	}

	if req.LocalBranch != "" {
		// TODO: Remove hard coded remote name
		if err := s.Git.DeleteRemoteTrackingBranch("origin", req.LocalBranch); err != nil {
			return nil, err
		}
	}

	return &res, nil
}
