package editor

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// Basic is an editor without special abilities. We can't detect whether the
// user saved the file or not.
type Basic struct {
	name string
	path string
}

// NewBasic builds a new basic editor.
func NewBasic(name string) (*Basic, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("could not resolve editor %q: %v", name, err)
	}

	return &Basic{name: name, path: path}, nil
}

// Name returns the name of the editor.
func (e *Basic) Name() string {
	return e.name
}

// EditString asks the user to edit the given string inside the editor.
func (e *Basic) EditString(s string) (string, error) {
	file, err := tempFileWithContents(s)
	if err != nil {
		return "", err
	}
	defer os.Remove(file)

	cmd := exec.Command(e.path, file)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %v", e.name, err)
	}

	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("could not read temporary file: %v", err)
	}

	return string(contents), nil
}
