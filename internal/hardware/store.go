package hardware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

type Store interface {
	GetAll(ctx context.Context) ([]Hardware, error)
	Create(ctx context.Context, newHardware NewHardware) (Hardware, error)
	Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error)
	Delete(ctx context.Context, hardwareID int) (Hardware, error)
}

const HarwareColumns = "id, name, brand, description, purchase_date, status, created_at, updated_at"

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

var getAllQuery = fmt.Sprintf("SELECT %s FROM hardware", HarwareColumns)

func (sqls *SQLiteStore) GetAll(ctx context.Context) ([]Hardware, error) {
	res := make([]Hardware, 0)
	if err := sqlscan.Select(ctx, sqls.client, &res, getAllQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

var createQuery = fmt.Sprintf(
	`INSERT INTO hardware (name, brand, description, purchase_date)
	 VALUES ($1, $2, $3, $4)
	 RETURNING %s`,
	HarwareColumns,
)

func (sqls *SQLiteStore) Create(ctx context.Context, newHardware NewHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, sqls.client, &res, createQuery,
		newHardware.Name,
		newHardware.Brand,
		newHardware.Description,
		newHardware.PurchaseDate,
	); err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

var updateQuery = fmt.Sprintf(
	`UPDATE hardware
	 SET name          = COALESCE($1, name),
		 brand         = COALESCE($2, brand),
		 description   = COALESCE($3, description),
		 purchase_date = COALESCE($4, purchase_date)
	 WHERE id = $5
	 RETURNING %s`,
	HarwareColumns,
)

func (sqlc *SQLiteStore) Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, sqlc.client, &res, updateQuery,
		updatedHardware.Name,
		updatedHardware.Brand,
		updatedHardware.Description,
		updatedHardware.PurchaseDate,
		updatedHardware.ID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware: %w", errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

var deleteQuery = fmt.Sprintf(
	`DELETE FROM hardware 
	 WHERE id = $1 
	 RETURNING %s`,
	HarwareColumns,
)

func (sqls *SQLiteStore) Delete(ctx context.Context, hardwareID int) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, sqls.client, &res, deleteQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware: %w", errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}
