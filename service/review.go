package service

import "context"

// ReviewStatus indicates whether a pull request has been reviewed
// successfully.
type ReviewStatus struct {
	// List of users who approved the pull request.
	Approvers []string

	// List of users who requested changes to the pull request.
	ChangedRequestedBy []string
}

// Review is a serice that provides access to pull request reviews.
type Review interface {
	// Check if a pull request has been approved for landing by reviewers.
	ReviewStatus(ctx context.Context, number int) (*ReviewStatus, error)
}
