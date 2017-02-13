package repo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abhinav/git-fu/git"
)

const (
	sshRemotePrefix  = "git@github.com:"
	httpRemotePrefix = "https://github.com/"
)

// Guess determines the Repo name based on the current Git repository's remotes.
func Guess(dir string) (*Repo, error) {
	newDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("could not resolve absolute path to %q: %v", dir, err)
	}
	dir = newDir

	url, err := git.Output("remote", "get-url", "origin")
	if err != nil {
		return nil, err
	}
	url = strings.TrimSpace(url)

	switch {
	case strings.HasPrefix(url, sshRemotePrefix):
		url = url[len(sshRemotePrefix):]
	case strings.HasPrefix(url, httpRemotePrefix):
		url = url[len(httpRemotePrefix):]
	default:
		return nil, fmt.Errorf(
			`remote "origin" (%v) of %q is not a GitHub remote`, url, dir)
	}

	return Parse(strings.TrimSuffix(url, ".git"))
}
