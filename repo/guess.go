package repo

import (
	"fmt"
	"strings"

	"github.com/abhinav/git-pr/gateway"
)

var _prefixes = []string{
	"ssh://git@github.com/",
	"git@github.com:",
	"https://github.com/",
}

// Guess determines the Repo name based on the current Git repository's remotes.
func Guess(git gateway.Git) (*Repo, error) {
	url, err := git.RemoteURL("origin")
	if err != nil {
		return nil, err
	}

	for _, prefix := range _prefixes {
		if strings.HasPrefix(url, prefix) {
			url := strings.TrimPrefix(url, prefix)
			return Parse(strings.TrimSuffix(url, ".git"))
		}
	}

	return nil, fmt.Errorf(`remote "origin" (%v) is not a GitHub remote`, url)
}
