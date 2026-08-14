package errors

import "errors"

var (
	ErrStoreInternal = errors.New("internal store error")
	ErrStoreNotFound = errors.New("not found")
	ErrStoreConflict = errors.New("conflicting state")
)
