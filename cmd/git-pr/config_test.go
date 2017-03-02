package main

import (
	"github.com/abhinav/git-fu/cli/clitest"
	"github.com/abhinav/git-fu/service"
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
