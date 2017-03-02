package main

import (
	"github.com/abhinav/git-fu/cli"
	"github.com/abhinav/git-fu/pr"
	"github.com/abhinav/git-fu/service"
)

// Common config for git-pr commands.
type config struct {
	cli.Config

	Service service.PR
}

type configBuilder func() (config, error)

func newConfigBuilder(cb cli.ConfigBuilder) configBuilder {
	return func() (config, error) {
		cfg, err := cb()
		if err != nil {
			return config{}, err
		}

		return config{
			Config: cfg,
			Service: &pr.Service{
				GitHub: cfg.GitHub(),
				Git:    cfg.Git(),
			},
		}, nil
	}
}
