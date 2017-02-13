package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/abhinav/git-fu/repo"

	"github.com/google/go-github/github"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const _keyringServiceName = "git-fu"

// Config is the common configuration for all programs in this package.
type Config interface {
	Repo() *repo.Repo
	GitHub() *github.Client
}

// ConfigBuilder builds a configuration lazily.
type ConfigBuilder func() (Config, error)

type globalConfig struct {
	RepoName    string `short:"r" long:"repo" value-name:"OWNER/REPO" description:"Name of the GitHub repository in the format 'owner/repo'. Defaults to the repository for the current directory."`
	GitHubUser  string `short:"u" long:"user" value-name:"USERNAME" env:"GITHUB_USER" required:"yes" description:"GitHub username."`
	GitHubToken string `short:"t" long:"token" env:"GITHUB_TOKEN" value-name:"TOKEN" description:"GitHub token used to make requests."`

	token        string
	repo         *repo.Repo
	httpClient   *http.Client
	githubClient *github.Client
}

var _ Config = (*globalConfig)(nil)

func (g *globalConfig) Token() (string, error) {
	switch {
	case g.token != "":
		return g.token, nil
	case g.GitHubToken != "":
		g.token = g.GitHubToken
		return g.GitHubToken, nil
	}

	var err error
	g.token, err = keyring.Get(_keyringServiceName, g.GitHubUser)
	switch err {
	case nil:
		return g.token, nil
	case keyring.ErrNotFound:
		return g.askForToken()
	default:
		return "", fmt.Errorf("failed to retrieve GitHub token from keyring: %v", err)
	}
}

func (g *globalConfig) askForToken() (string, error) {
	fmt.Println("GitHub token not found. " +
		"Please generate one at https://github.com/settings/tokens")
	fmt.Printf("GitHub token for %v: ", g.GitHubUser)
	if _, err := fmt.Scanln(&g.token); err != nil {
		return "", err
	}

	g.token = strings.TrimSpace(g.token)
	if g.token == "" {
		return "", fmt.Errorf("GitHub token cannot be blank")
	}

	// TODO: verify token validity before storing

	if err := keyring.Set(_keyringServiceName, g.GitHubUser, g.token); err != nil {
		return "", fmt.Errorf("failed to store GitHub token in keyring: %v", err)
	}

	return g.token, nil
}

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

	token, err := g.Token()
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	g.httpClient = oauth2.NewClient(context.Background(), tokenSource)
	g.githubClient = github.NewClient(g.httpClient)
	return g, nil
}

func (g *globalConfig) Repo() *repo.Repo {
	return g.repo
}

func (g *globalConfig) GitHub() *github.Client {
	return g.githubClient
}
