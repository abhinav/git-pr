package pr

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/service"

	"github.com/google/go-github/github"
	"go.uber.org/multierr"
)

func (s *Service) checkApproved(ctx context.Context, pr *github.PullRequest) error {
	reviews, err := s.GitHub.ListPullRequestReviews(ctx, pr.GetNumber())
	if err != nil {
		return err
	}

	for _, review := range reviews {
		if review.Status != gateway.PullRequestChangesRequested {
			continue
		}

		err = multierr.Append(err,
			fmt.Errorf("%v has requested changes on %v", review.User, pr.GetHTMLURL()))
	}

	return err
}

func (s *Service) checkBuilt(ctx context.Context, ref string) error {
	status, err := s.GitHub.GetBuildStatus(ctx, ref)
	if err != nil {
		return err
	}

	if status.State == gateway.BuildSuccess {
		return nil
	}

	for _, s := range status.Statuses {
		var msg string
		switch s.State {
		case gateway.BuildSuccess:
			continue
		case gateway.BuildPending:
			msg = fmt.Sprintf("%v is still running", s.Name)
		default:
			msg = fmt.Sprintf("%v state is %v: %v", s.Name, s.State, s.Message)
		}
		err = multierr.Append(err, errors.New(msg))
	}

	return err
}

func (s *Service) checkLandable(ctx context.Context, pr *github.PullRequest) error {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		errors error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := s.checkApproved(ctx, pr); err != nil {
			mu.Lock()
			errors = multierr.Append(errors, err)
			mu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := s.checkBuilt(ctx, pr.Head.GetRef()); err != nil {
			mu.Lock()
			errors = multierr.Append(errors, err)
			mu.Unlock()
		}
	}()

	wg.Wait()
	return errors
}

// Land the given pull request.
func (s *Service) Land(ctx context.Context, req *service.LandRequest) (*service.LandResponse, error) {
	pr := req.PullRequest
	if !req.NoCheck {
		if err := s.checkLandable(ctx, pr); err != nil {
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
