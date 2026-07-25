package mapper

import (
	"github.com/jinzhu/copier"
)

// Map maps fields from source to destination.
// Destination must be a pointer to a struct.
// It uses github.com/jinzhu/copier under the hood.
func Map(dst interface{}, src interface{}) error {
	return copier.Copy(dst, src)
}

// MapList maps a slice of source objects to a slice of destination objects.
// Destination must be a pointer to a slice.
func MapList(dst interface{}, src interface{}) error {
	return copier.Copy(dst, src)
}
