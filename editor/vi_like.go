package editor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// ErrFileUnsaved is returned if the user exited a vi-like editor without
// saving the file.
var ErrFileUnsaved = errors.New("file was not saved")

const _viLikeplaceholder = "#### git-stack placeholder ####"

// ViLike is an editor backed by vi/vim/neovim/etc.
//
// If the user exits the editor without saving the file, no changes are
// recorded.
type ViLike struct {
	name string
	path string
}

// NewViLike builds a new vi-like editor.
func NewViLike(name string) (*ViLike, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("could not resolve editor %q: %v", name, err)
	}

	return &ViLike{name: name, path: path}, nil
}

// Name of the editor.
func (vi *ViLike) Name() string {
	return vi.name
}

// EditString asks the user to edit the given string inside a vi-like editor.
// ErrFileUnsaved is returned if the user exits the editor without saving the
// file.
func (vi *ViLike) EditString(in string) (string, error) {
	sourceFile, err := tempFileWithContents(in)
	if err != nil {
		return "", err
	}
	defer os.Remove(sourceFile)

	destFile, err := tempFileWithContents(_viLikeplaceholder)
	if err != nil {
		return "", err
	}
	defer os.Remove(destFile)

	// To detect this, we create a file with some placeholder contents, load
	// it up in vim and replace the placeholder contents with the actual
	// contents we want to edit. If the user doesn't save it, the placeholder
	// will be retained.
	cmd := exec.Command(
		vi.path,
		"-c", "%d", // delete placeholder
		"-c", "0read "+sourceFile, // read the source file
		"-c", "$d", // delete trailing newline
		"-c", "set ft=gitcommit | 0", // set filetype and go to start of file
		destFile)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %v", vi.name, err)
	}

	contents, err := ioutil.ReadFile(destFile)
	if err != nil {
		return "", fmt.Errorf("could not read temporary file: %v", err)
	}

	out := string(contents)
	if out == _viLikeplaceholder {
		return "", ErrFileUnsaved
	}

	return out, nil
}
