package errors

import "errors"

var (
	ErrStoreInternal = errors.New("internal store error")
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflicting state")
)
