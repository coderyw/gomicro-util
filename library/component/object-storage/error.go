package object_storage

import "errors"

var (
	ErrNotFound = errors.New("object not found")
	ErrDenied   = errors.New("permission denied")
	ErrInternal = errors.New("storage internal error")
)
