package git

import (
	"bytes"
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
