package git

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/abhinav/git-fu/gateway"
	"github.com/stretchr/testify/require"
)

func TestPushNoRefs(t *testing.T) {
	dir, err := ioutil.TempDir("", "git-pr")
	require.NoError(t, err, "couldn't create a temporary directory")
	defer os.RemoveAll(dir)

	restore, err := chdir(dir)
	require.NoError(t, err, "could not cd into %v", dir)
	defer restore()

	require.NoError(t, exec.Command("git", "init").Run(),
		"failed to set up git repo")

	gw, err := NewGateway(dir)
	require.NoError(t, err, "could not set up gateway")

	// We don't have any remotes but that isn't a problem simply because this
	// operation shouldn't do *anything* at all
	err = gw.Push(&gateway.PushRequest{
		Remote: "origin",
		Refs:   make(map[string]string),
		Force:  true,
	})
	require.NoError(t, err)
}

func chdir(dir string) (restore func(), _ error) {
	oldDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(dir); err != nil {
		return nil, err
	}

	return func() { os.Chdir(oldDir) }, nil
}
