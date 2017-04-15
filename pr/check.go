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

// LandCheck verifies that a pull request can be landed.
type LandCheck interface {
	Check(context.Context, *github.PullRequest) error
}

// ApprovalCheck is a LandCheck that fails if a pull request has not been
// approved to land.
type ApprovalCheck struct{ Review service.Review }

// Check that the pull request has been approved.
func (c *ApprovalCheck) Check(ctx context.Context, pr *github.PullRequest) error {
	status, err := c.Review.ReviewStatus(ctx, pr.GetNumber())
	if err != nil {
		return err
	}

	if len(status.Approvers) == 0 {
		err = multierr.Append(err,
			fmt.Errorf("%v has not been approved by anyone", pr.GetHTMLURL()))
	}

	for _, u := range status.ChangedRequestedBy {
		err = multierr.Append(err,
			fmt.Errorf("%v has requested changes on %v", u, pr.GetHTMLURL()))
	}

	return err
}

// BuildCheck is a LandCheck that fails if the HEAD commit in a pull request
// has not passed all build checks.
type BuildCheck struct{ GitHub gateway.GitHub }

// Check that the pull request has passed all build checks.
func (c *BuildCheck) Check(ctx context.Context, pr *github.PullRequest) error {
	status, err := c.GitHub.GetBuildStatus(ctx, pr.Head.GetRef())
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

// MultiLandCheck runs multiple LandChecks concurrently.
type MultiLandCheck []LandCheck

// Check the pull request using all associated LandChecks.
func (ml MultiLandCheck) Check(ctx context.Context, pr *github.PullRequest) error {
	var (
		err error
		wg  sync.WaitGroup
		mu  sync.Mutex
	)

	for _, lc := range ml {
		wg.Add(1)
		go func(lc LandCheck) {
			defer wg.Done()
			if e := lc.Check(ctx, pr); e != nil {
				mu.Lock()
				err = multierr.Append(err, e)
				mu.Unlock()
			}
		}(lc)
	}

	return err
}
