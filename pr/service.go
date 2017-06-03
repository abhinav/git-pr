package pr

import (
	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/service"
)

// ServiceConfig specifies the different parameters for a PR service.
type ServiceConfig struct {
	GitHub gateway.GitHub
	Git    gateway.Git
}

// Service is a PR service.
type Service struct {
	gh  gateway.GitHub
	git gateway.Git

	// Hidden option to customize how we rebase pull requests.
	rebasePullRequests func(rebasePRConfig) (map[int]rebasedPullRequest, error)
}

// NewService builds a new PR service with the given configuration.
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		gh:                 cfg.GitHub,
		git:                cfg.Git,
		rebasePullRequests: rebasePullRequests,
	}
}

var _ service.PR = (*Service)(nil)
