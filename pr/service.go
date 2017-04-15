package pr

import (
	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/service"
)

// Service TODO
type Service struct {
	GitHub    gateway.GitHub
	Git       gateway.Git
	LandCheck LandCheck
}

var _ service.PR = (*Service)(nil)
