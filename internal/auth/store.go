package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

type Store interface {
	GetUserWithCreds(ctx context.Context, email string) (UserWithCreds, error)
}

var _ Store = (*SQLiteStore)(nil)

type SQLiteStore struct {
	client *sql.DB
}

type SQLiteStoreOptions struct {
	Client *sql.DB
}

func NewSQLiteStore(opts *SQLiteStoreOptions) (*SQLiteStore, error) {
	if opts == nil {
		return nil, fmt.Errorf("SQLiteStoreOptions cannot be nil")
	}
	if opts.Client == nil {
		return nil, fmt.Errorf("SQLiteStoreOptions.Client cannot be nil")
	}

	store := &SQLiteStore{
		client: opts.Client,
	}

	return store, nil
}

var getUserWithCredsQuery = `
SELECT id, email, full_name, role, password_hash
FROM profiles
WHERE email = $1
`

func (sqls *SQLiteStore) GetUserWithCreds(ctx context.Context, email string) (UserWithCreds, error) {
	var res UserWithCreds
	if err := sqlscan.Get(ctx, sqls.client, &res, getUserWithCredsQuery, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserWithCreds{}, fmt.Errorf("user: %w", errutils.ErrStoreNotFound)
		}

		return UserWithCreds{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}
