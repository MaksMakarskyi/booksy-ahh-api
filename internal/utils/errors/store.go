package errors

import (
	"errors"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	ErrStoreInternal  = errors.New("internal store error")
	ErrStoreNotFound  = errors.New("not found")
	ErrStoreConflict  = errors.New("conflicting state")
	ErrStoreForbidden = errors.New("operation is forbidden")
	ErrStoreTxBegin   = errors.New("failed to begin transaction")
	ErrStoreTxCommit  = errors.New("failed to commit transaction")
)

// IsForeignKeyViolation reports whether err is SQLite refusing a write because
// another row still references it — an ON DELETE RESTRICT, in practice.
//
// Like a duplicate key, this is the caller's problem (409), not ours (500):
// "this employee still has rental history" is something the client can act on.
//
// Both extended codes are matched deliberately. SQLite implements RESTRICT
// through its trigger machinery, so the driver reports 1811
// (SQLITE_CONSTRAINT_TRIGGER) rather than 787 (SQLITE_CONSTRAINT_FOREIGNKEY)
// even though the message reads "FOREIGN KEY constraint failed". Matching only
// the obvious constant would never fire.
func IsForeignKeyViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	switch sqliteErr.Code() {
	case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY, sqlite3.SQLITE_CONSTRAINT_TRIGGER:
		return true
	default:
		return false
	}
}

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
