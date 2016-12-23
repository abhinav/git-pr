package editor

// Editor allows editing strings.
type Editor interface {
	EditString(string) (string, error)
}
