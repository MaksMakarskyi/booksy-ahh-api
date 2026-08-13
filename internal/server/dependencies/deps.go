package dependencies

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
)

type Registry struct {
	DB *sql.DB

	Config *config.Config
}

func NewRegistry(ctx context.Context, cfg *config.Config) (*Registry, error) {
	// Database
	db, err := sql.Open("sqlite", cfg.DatabaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to establish db connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Registry
	registry := Registry{
		DB:     db,
		Config: cfg,
	}

	return &registry, nil
}

func (r *Registry) Close() error {
	if err := r.DB.Close(); err != nil {
		return fmt.Errorf("failed to close db: %w", err)
	}

	return nil
}
