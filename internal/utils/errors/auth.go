package errors

import "errors"

var (
	ErrGeneratePasswordHash = errors.New("failed to generate hash for password")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidBearerToken   = errors.New("invalid bearer credentials")
	ErrUserNotFound         = errors.New("user not found")
)
