package export

import "errors"

// Errors returned by the export package.
var (
	// ErrNotSlice is returned by FromStructs when the input is not a slice.
	ErrNotSlice = errors.New("export: expected a slice of structs")
)
