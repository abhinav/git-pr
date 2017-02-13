package editor

// Pick an editor based on the given name.
func Pick(name string) (Editor, error) {
	switch name {
	case "vi", "vim", "nvim", "mvim", "gvim", "elvis":
		return NewViLike(name)
	default:
		return NewBasic(name)
	}
}
