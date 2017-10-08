package clitest

import (
	"github.com/abhinav/git-pr/cli"
	"github.com/abhinav/git-pr/gateway"
	"github.com/abhinav/git-pr/repo"
)

// ConfigBuilder may be used to build a cli.Config from static values.
type ConfigBuilder struct {
	Git        gateway.Git
	Repo       *repo.Repo
	GitHub     gateway.GitHub
	GitHubUser string
}

// Build the cli.Config. This function may also be used as a
// cli.ConfigBuilder.
func (c *ConfigBuilder) Build() (cli.Config, error) {
	// We never return an error. It's used only to satisfy the
	// cli.ConfigBuilder signature.
	return &config{*c}, nil
}

type config struct{ data ConfigBuilder }

func (c *config) Git() gateway.Git {
	return c.data.Git
}

func (c *config) Repo() *repo.Repo {
	return c.data.Repo
}

func (c *config) CurrentGitHubUser() string {
	return c.data.GitHubUser
}

func (c *config) GitHub() gateway.GitHub {
	return c.data.GitHub
}
