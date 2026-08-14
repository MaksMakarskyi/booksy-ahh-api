package profiles

import (
	"context"
	"database/sql"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

type Store interface {
	Create(ctx context.Context, record NewProfile) (Profile, error)
}

const ProfileColumns = "id, email, full_name, role, created_at, updated_at"

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

var createQuery = fmt.Sprintf(
	`INSERT INTO profiles (email, full_name, role, password_hash)
	 VALUES ($1, $2, $3, $4)
	 RETURNING %s`,
	ProfileColumns,
)

func (sqls *SQLiteStore) Create(ctx context.Context, record NewProfile) (Profile, error) {
	var res Profile
	if err := sqlscan.Get(ctx, sqls.client, &res, createQuery,
		record.Email,
		record.FullName,
		record.Role,
		record.PasswordHash,
	); err != nil {
		if errutils.IsUniqueViolation(err) {
			return Profile{}, fmt.Errorf("profile email: %w", errutils.ErrStoreConflict)
		}

		return Profile{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}
