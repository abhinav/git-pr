package editor

// Editor allows editing strings.
type Editor interface {
	EditString(string) (string, error)
}

//go:generate mockgen -package=editortest -destination=editortest/mocks.go github.com/abhinav/git-pr/editor Editor
