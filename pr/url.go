package pr

import (
	"fmt"

	"github.com/google/go-github/github"
)

// URL returns the URL for the given pull request.
func URL(pr *github.PullRequest) string {
	repo := pr.Base.Repo
	return fmt.Sprintf(
		"https://github.com/%v/%v/pull/%v", *repo.Owner.Login, *repo.Name, *pr.Number)
}
