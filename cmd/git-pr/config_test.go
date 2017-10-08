package main

import (
	"github.com/abhinav/git-pr/cli/clitest"
	"github.com/abhinav/git-pr/service"
)

type fakeConfigBuilder struct {
	clitest.ConfigBuilder

	Service service.PR
}

func (f *fakeConfigBuilder) Build() (config, error) {
	c, err := f.ConfigBuilder.Build()
	if err != nil {
		return config{}, err
	}

	return config{Config: c, Service: f.Service}, nil
}
