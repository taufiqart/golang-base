package pdf

import "errors"

// ErrNoColumns is returned when a table has no columns defined.
var ErrNoColumns = errors.New("pdf: table has no columns")
