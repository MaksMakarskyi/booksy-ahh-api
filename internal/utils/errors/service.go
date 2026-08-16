package errors

import "errors"

var (
	ErrServiceInternal = errors.New("internal service failed")
	ErrServiceExternal = errors.New("external service failed")
)
