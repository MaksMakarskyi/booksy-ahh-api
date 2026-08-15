package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

const (
	profileColumns = "id, email, full_name, role, created_at, updated_at"

	getAllQuery = `
SELECT ` + profileColumns + `
FROM profiles
WHERE role = 'employee' OR id = $1`

	createQuery = `
INSERT INTO profiles (email, full_name, role, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING ` + profileColumns

	deleteQuery = `
DELETE FROM profiles
WHERE id = $1 AND role = 'employee'
RETURNING ` + profileColumns

	createAdminIfAbsentQuery = `
INSERT INTO profiles (email, full_name, role, password_hash)
VALUES ($1, $2, 'admin', $3)
ON CONFLICT (email) DO NOTHING
RETURNING ` + profileColumns
)

type Store interface {
	GetAll(ctx context.Context, userID int) ([]Profile, error)
	Create(ctx context.Context, record NewProfile) (Profile, error)
	Delete(ctx context.Context, userID int, profileID int) (Profile, error)

	CreateAdminIfAbsent(ctx context.Context, record NewProfile) (created bool, err error)
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

	return &SQLiteStore{client: opts.Client}, nil
}

func (s *SQLiteStore) GetAll(ctx context.Context, userID int) ([]Profile, error) {
	res := make([]Profile, 0)
	if err := sqlscan.Select(ctx, s.client, &res, getAllQuery, userID); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Create(ctx context.Context, record NewProfile) (Profile, error) {
	var res Profile
	if err := sqlscan.Get(ctx, s.client, &res, createQuery,
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

func (s *SQLiteStore) Delete(ctx context.Context, userID int, profileID int) (Profile, error) {
	if userID == profileID {
		return Profile{}, fmt.Errorf("%w: users cannot remove themselves", errutils.ErrStoreConflict)
	}

	var res Profile
	if err := sqlscan.Get(ctx, s.client, &res, deleteQuery, profileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, fmt.Errorf("profile: %w", errutils.ErrStoreNotFound)
		}
		if errutils.IsForeignKeyViolation(err) {
			return Profile{}, fmt.Errorf(
				"%w: profile %d still has rental history", errutils.ErrStoreConflict, profileID,
			)
		}

		return Profile{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) CreateAdminIfAbsent(ctx context.Context, record NewProfile) (bool, error) {
	var res Profile
	err := sqlscan.Get(ctx, s.client, &res, createAdminIfAbsentQuery,
		record.Email,
		record.FullName,
		record.PasswordHash,
	)
	if err != nil {
		// No row returned means the INSERT hit the conflict clause: an account
		// with this email already exists, which is the normal path on every
		// restart after the first.
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return true, nil
}
