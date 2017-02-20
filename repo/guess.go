package repo

import (
	"fmt"
	"strings"

	"github.com/abhinav/git-fu/gateway"
)

const (
	sshRemotePrefix  = "git@github.com:"
	httpRemotePrefix = "https://github.com/"
)

// Guess determines the Repo name based on the current Git repository's remotes.
func Guess(git gateway.Git) (*Repo, error) {
	url, err := git.RemoteURL("origin")
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(url, sshRemotePrefix):
		url = url[len(sshRemotePrefix):]
	case strings.HasPrefix(url, httpRemotePrefix):
		url = url[len(httpRemotePrefix):]
	default:
		return nil, fmt.Errorf(`remote "origin" (%v) is not a GitHub remote`, url)
	}

	return Parse(strings.TrimSuffix(url, ".git"))
}
