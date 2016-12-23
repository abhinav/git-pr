package pr

import (
	"errors"
	"fmt"

	"github.com/abhinav/git-fu/editor"

	"github.com/google/go-github/github"
)

// Lander lands GitHub pull requests
type Lander interface {
	Land(*github.PullRequest) error
}

// MessageEditLander allows editing the commit message before landing a PR.
type MessageEditLander struct {
	Lander Lander
	Editor editor.Editor
}

// Land a PR, modifying its commit body using the Editor first.
func (l *MessageEditLander) Land(pr *github.PullRequest) error {
	// TODO Use arc-style title-body separation in interactive editor to allow
	// customizing the title as well.
	body, err := l.Editor.EditString(*pr.Body)
	if err != nil {
		return err
	}
	pr.Body = &body
	return l.Lander.Land(pr)
}

// SquashLander lands a PR as a single squashed commit.
type SquashLander struct {
	GitHubClient *github.Client
}

// Land the given PR, squashing all commits into one.
//
// The PR description will be used as the commit body.
func (l *SquashLander) Land(pr *github.PullRequest) error {
	repo := pr.Base.Repo
	result, _, err := l.GitHubClient.PullRequests.Merge(
		*repo.Owner.Login, *repo.Name, *pr.Number, *pr.Body,
		&github.PullRequestOptions{
			CommitTitle: fmt.Sprintf("%v (#%v)", *pr.Title, *pr.Number),
			MergeMethod: "squash",
		})
	if err != nil {
		return err
	}

	if result.Merged == nil || !*result.Merged {
		return errors.New(*result.Message)
	}
	return nil
}
