package hardware

import (
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
)

// Build builds new handler from dependencies
func Build(deps *dependencies.Registry) (*Handler, error) {
	store, err := NewSQLiteStore(&SQLiteStoreOptions{
		Client: deps.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build store: %w", err)
	}

	handler, err := NewHandler(&HandlerOptions{
		Store: store,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build handler: %w", err)
	}

	return handler, nil
}
