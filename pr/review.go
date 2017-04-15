package pr

import (
	"context"
	"sort"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/service"
)

// ReviewService is a pull request review service.
type ReviewService struct {
	GitHub gateway.GitHub
}

var _ service.Review = (*ReviewService)(nil)

// ReviewStatus checks whether the given pull request has been approved to
// land.
func (s *ReviewService) ReviewStatus(ctx context.Context, number int) (*service.ReviewStatus, error) {
	reviews, err := s.GitHub.ListPullRequestReviews(ctx, number)
	if err != nil {
		return nil, err
	}

	approvers := make(map[string]struct{})
	changesRequested := make(map[string]struct{})
	for _, review := range reviews {
		user := review.User

		// There can be multiple reviews by the same user. The reviews are
		// in-order so we should only consider the latest one.
		delete(approvers, user)
		delete(changesRequested, user)

		switch review.Status {
		case gateway.PullRequestChangesRequested:
			changesRequested[user] = struct{}{}
		case gateway.PullRequestApproved:
			approvers[user] = struct{}{}
		}
	}

	var result service.ReviewStatus

	for u := range approvers {
		result.Approvers = append(result.Approvers, u)
	}

	for u := range changesRequested {
		result.ChangedRequestedBy = append(result.ChangedRequestedBy, u)
	}

	sort.Strings(result.Approvers)
	sort.Strings(result.ChangedRequestedBy)
	return &result, nil
}
