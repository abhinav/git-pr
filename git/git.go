package git

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// Output runs the git with the given arguments and returns the output.
func Output(args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	return stdout.String(), err
}

// Run the given git command.
func Run(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// ApplyPatches applies a series of patches. This relies on the `git-am`
// command. Remaining args are passed to the `git-am` command.
func ApplyPatches(patch io.Reader, args ...string) error {
	cmd := exec.Command("git", append([]string{"am"}, args...)...)
	cmd.Stdin = patch
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
