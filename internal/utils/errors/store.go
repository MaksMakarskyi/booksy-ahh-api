package errors

import (
	"errors"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	ErrStoreInternal = errors.New("internal store error")
	ErrStoreNotFound = errors.New("not found")
	ErrStoreConflict = errors.New("conflicting state")
)

func IsUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	switch sqliteErr.Code() {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
		return true
	default:
		return false
	}
}
