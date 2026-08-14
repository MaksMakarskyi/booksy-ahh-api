package dependencies

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	jwtutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/jwt"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type Registry struct {
	// Main dependencies
	DB        *sql.DB
	JWTIssuer *jwtutils.Issuer
	Config    *config.Config

	// Echo dependencies
	Validator      echo.Validator
	ErrorHandler   echo.HTTPErrorHandler
	JSONSerializer echo.JSONSerializer
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

	jwtIssuer, err := jwtutils.NewIssuer(&jwtutils.IssuerOptions{
		Secret: cfg.JWTSecret,
		TTL:    cfg.JWTTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build token issuer: %w", err)
	}

	// Echo dependencies
	val := valutils.NewCustomValidator()

	errorHandler, err := errutils.NewErrorHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build error handler: %w", err)
	}

	jsonSerializer := valutils.StrictJSONSerializer{}

	// Registry
	registry := Registry{
		DB:        db,
		JWTIssuer: jwtIssuer,
		Config:    cfg,

		Validator:      val,
		ErrorHandler:   errorHandler,
		JSONSerializer: jsonSerializer,
	}

	return &registry, nil
}

func (r *Registry) Close() error {
	if err := r.DB.Close(); err != nil {
		return fmt.Errorf("failed to close db: %w", err)
	}

	return nil
}
