package hardware

import (
	"context"
	"database/sql"
	"fmt"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

type Store interface {
	GetAll(ctx context.Context) ([]Hardware, error)
	Create(ctx context.Context, newHardware NewHardware) (Hardware, error)
}

const HarwareColumns = "id, name, brand, description, purchase_date, status, created_at, updated_at"

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
