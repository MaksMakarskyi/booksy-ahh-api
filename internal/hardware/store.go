package hardware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

const (
	hardwareColumns = "id, name, brand, description, purchase_date, status, created_at, updated_at"

	getAllQuery = `
SELECT ` + hardwareColumns + `
FROM hardware`

	createQuery = `
INSERT INTO hardware (name, brand, description, purchase_date)
VALUES ($1, $2, $3, $4)
RETURNING ` + hardwareColumns

	updateQuery = `
UPDATE hardware
SET name          = COALESCE($1, name),
    brand         = COALESCE($2, brand),
    description   = COALESCE($3, description),
    purchase_date = COALESCE($4, purchase_date)
WHERE id = $5
RETURNING ` + hardwareColumns

	deleteQuery = `
DELETE FROM hardware
WHERE id = $1
RETURNING ` + hardwareColumns
)

type Store interface {
	GetAll(ctx context.Context) ([]Hardware, error)
	Create(ctx context.Context, newHardware NewHardware) (Hardware, error)
	Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error)
	Delete(ctx context.Context, hardwareID int) (Hardware, error)
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

func (s *SQLiteStore) GetAll(ctx context.Context) ([]Hardware, error) {
	res := make([]Hardware, 0)
	if err := sqlscan.Select(ctx, s.client, &res, getAllQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Create(ctx context.Context, newHardware NewHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, createQuery,
		newHardware.Name,
		newHardware.Brand,
		newHardware.Description,
		newHardware.PurchaseDate,
	); err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, updateQuery,
		updatedHardware.Name,
		updatedHardware.Brand,
		updatedHardware.Description,
		updatedHardware.PurchaseDate,
		updatedHardware.ID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", updatedHardware.ID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, hardwareID int) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, deleteQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}
