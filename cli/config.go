package cli

import (
	"context"
	"net/http"

	"github.com/abhinav/git-fu/entity"
	"github.com/abhinav/git-fu/repo"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// Config is the common configuration for all programs in this package.
type Config interface {
	Repo() *entity.Repo
	GitHub() *github.Client
}

// ConfigBuilder builds a configuration lazily.
type ConfigBuilder func() (Config, error)

type globalConfig struct {
	RepoName    string `short:"r" long:"repo" value-name:"OWNER/REPO" description:"Name of the GitHub repository in the format 'owner/repo'. Defaults to the repository for the current directory."`
	GitHubToken string `short:"t" long:"token" env:"GITHUB_TOKEN" value-name:"TOKEN" required:"yes" description:"GitHub token used to make requests."`

	repo         *entity.Repo
	httpClient   *http.Client
	githubClient *github.Client
}

var _ Config = (*globalConfig)(nil)

// globalConfig.Build is a ConfigBuilder
func (g *globalConfig) Build() (_ Config, err error) {
	if g.RepoName != "" {
		g.repo, err = repo.Parse(g.RepoName)
	} else {
		g.repo, err = repo.Guess(".")
	}
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: g.GitHubToken})
	g.httpClient = oauth2.NewClient(context.Background(), tokenSource)
	g.githubClient = github.NewClient(g.httpClient)

	return g, nil
}

func (g *globalConfig) Repo() *entity.Repo {
	return g.repo
}

func (g *globalConfig) GitHub() *github.Client {
	return g.githubClient
}
