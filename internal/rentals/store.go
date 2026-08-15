package rentals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

const (
	rentalColumns = "id, hardware_id, user_id, rented_at, returned_at"

	getAllQuery = `
SELECT r.id          AS id,
       r.user_id     AS user_id,
       h.id          AS hardware_id,
       h.name        AS name,
       h.brand       AS brand,
       h.description AS description,
       r.rented_at   AS rented_at,
       r.returned_at AS returned_at
FROM rentals r
INNER JOIN hardware h ON h.id = r.hardware_id
WHERE r.user_id = $1
ORDER BY r.returned_at IS NOT NULL, r.rented_at DESC`

	selectHardwareStatusQuery = `
SELECT status
FROM hardware
WHERE id = $1`

	claimHardwareQuery = `
UPDATE hardware
SET status = 'in_use'
WHERE id = $1 AND status = 'available'`

	releaseHardwareQuery = `
UPDATE hardware
SET status = 'available'
WHERE id = $1 AND status = 'in_use'`

	createRentalQuery = `
INSERT INTO rentals (user_id, hardware_id)
VALUES ($1, $2)
RETURNING ` + rentalColumns

	selectRentalQuery = `
SELECT ` + rentalColumns + `
FROM rentals
WHERE id = $1`

	closeRentalQuery = `
UPDATE rentals
SET returned_at = $1
WHERE id = $2 AND returned_at IS NULL
RETURNING ` + rentalColumns
)

type Store interface {
	GetAll(ctx context.Context, userID int) ([]RentalWithHardware, error)
	Create(ctx context.Context, hardwareID int, userID int) (Rental, error)
	Return(ctx context.Context, rentalID int, userID int) (Rental, error)
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

func (s *SQLiteStore) GetAll(ctx context.Context, userID int) ([]RentalWithHardware, error) {
	res := make([]RentalWithHardware, 0)
	if err := sqlscan.Select(ctx, s.client, &res, getAllQuery, userID); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Create(ctx context.Context, hardwareID int, userID int) (Rental, error) {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxBegin, err)
	}
	defer tx.Rollback()

	var status string
	if err := sqlscan.Get(ctx, tx, &status, selectHardwareStatusQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Rental{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	claimed, err := tx.ExecContext(ctx, claimHardwareQuery, hardwareID)
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}
	affected, err := claimed.RowsAffected()
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}
	if affected == 0 {
		return Rental{}, fmt.Errorf(
			"%w: hardware %d is %s", errutils.ErrStoreConflict, hardwareID, status,
		)
	}

	var rental Rental
	if err := sqlscan.Get(ctx, tx, &rental, createRentalQuery, userID, hardwareID); err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	if err := tx.Commit(); err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxCommit, err)
	}

	return rental, nil
}

func (s *SQLiteStore) Return(ctx context.Context, rentalID int, userID int) (Rental, error) {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxBegin, err)
	}
	defer tx.Rollback()

	var existing Rental
	if err := sqlscan.Get(ctx, tx, &existing, selectRentalQuery, rentalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Rental{}, fmt.Errorf("rental %d: %w", rentalID, errutils.ErrStoreNotFound)
		}

		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	if existing.UserID != userID {
		return Rental{}, fmt.Errorf("rental %d: %w", rentalID, errutils.ErrStoreForbidden)
	}
	if existing.ReturnedAt != nil {
		return Rental{}, fmt.Errorf(
			"%w: rental %d was already returned", errutils.ErrStoreConflict, rentalID,
		)
	}

	var rental Rental
	err = sqlscan.Get(ctx, tx, &rental, closeRentalQuery,
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		rentalID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Rental{}, fmt.Errorf(
				"%w: rental %d was already returned", errutils.ErrStoreConflict, rentalID,
			)
		}

		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	released, err := tx.ExecContext(ctx, releaseHardwareQuery, rental.HardwareID)
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}
	releasedRows, err := released.RowsAffected()
	if err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}
	if releasedRows == 0 {
		return Rental{}, fmt.Errorf(
			"%w: hardware %d is not in use", errutils.ErrStoreConflict, rental.HardwareID,
		)
	}

	if err := tx.Commit(); err != nil {
		return Rental{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxCommit, err)
	}

	return rental, nil
}
