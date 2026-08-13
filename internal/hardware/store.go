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
}

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

func (sqls *SQLiteStore) GetAll(ctx context.Context) ([]Hardware, error) {
	q := `SELECT id, name, brand, description, purchase_date, status, created_at, updated_at
		  FROM hardware`

	hardwares := make([]Hardware, 0)
	if err := sqlscan.Select(ctx, sqls.client, &hardwares, q); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return hardwares, nil
}
