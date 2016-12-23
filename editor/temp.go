package editor

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// tempFileWithContents creates a new temporary file with the given contents
// and returns a path to it. The caller is responsible for deleting it.
func tempFileWithContents(contents string) (string, error) {
	f, err := ioutil.TempFile("", "git-stack-edit-string")
	if err != nil {
		return "", fmt.Errorf("could not open a temporary file: %v", err)
	}

	if _, err := io.WriteString(f, contents); err != nil {
		// TODO: combine errors
		f.Close()
		os.Remove(f.Name())
		return "", err
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}
