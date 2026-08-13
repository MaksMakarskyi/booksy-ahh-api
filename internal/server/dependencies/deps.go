package dependencies

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type Registry struct {
	// Main dependencies
	DB     *sql.DB
	Config *config.Config

	// Echo dependencies
	Validator    *valutils.CustomValidator
	ErrorHandler echo.HTTPErrorHandler
}

func NewRegistry(ctx context.Context, cfg *config.Config) (*Registry, error) {
	// Main dependencies
	db, err := sql.Open("sqlite", cfg.DatabaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to establish db connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Echo dependencies
	val := valutils.NewCustomValidator()

	errorHandler, err := errutils.NewErrorHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build error handler: %w", err)
	}

	// Registry
	registry := Registry{
		DB:     db,
		Config: cfg,

		Validator:    val,
		ErrorHandler: errorHandler,
	}

	return &registry, nil
}

func (r *Registry) Close() error {
	if err := r.DB.Close(); err != nil {
		return fmt.Errorf("failed to close db: %w", err)
	}

	return nil
}
