package ptr

// String returns a pointer to the given string
func String(x string) *string { return &x }
