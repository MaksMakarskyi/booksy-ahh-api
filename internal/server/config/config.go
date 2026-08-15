package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Env  AppEnv `env:"APP_ENV, default=production"`
	Port string `env:"PORT, default=8080"`

	DatabaseUrl string `env:"DATABASE_URL, required"`

	JWTSecret string        `env:"JWT_SECRET, required"`
	JWTTTL    time.Duration `env:"JWT_TTL, default=12h"`

	CORSOrigins []string `env:"CORS_ORIGINS, default=*"`

	GooseTable string `env:"GOOSE_TABLE, default=goose_migrations"`

	AdminEmail    string `env:"ADMIN_EMAIL, required"`
	AdminPassword string `env:"ADMIN_PASSWORD, required"`
	AdminName     string `env:"ADMIN_NAME, default=Administrator"`
}

const minJWTSecretBytes = 32

func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := new(Config)

	if err := envconfig.Process(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to load env vars: %w", err)
	}

	if !cfg.Env.IsValid() {
		return nil, fmt.Errorf("APP_ENV is not valid, got: %q", cfg.Env)
	}
	if strings.TrimSpace(cfg.DatabaseUrl) == "" {
		return nil, fmt.Errorf("DATABASE_URL must not be empty")
	}
	if len(cfg.JWTSecret) < minJWTSecretBytes {
		return nil, fmt.Errorf(
			"JWT_SECRET must be at least %d bytes, got %d",
			minJWTSecretBytes, len(cfg.JWTSecret),
		)
	}
	if cfg.JWTTTL <= 0 {
		return nil, fmt.Errorf("JWT_TTL must be positive, got %s", cfg.JWTTTL)
	}
	if len(cfg.CORSOrigins) == 0 {
		return nil, fmt.Errorf("CORS_ORIGINS must not be empty")
	}

	return cfg, nil
}

type AppEnv string

const (
	Development AppEnv = "development"
	Production  AppEnv = "production"
)

func (ae AppEnv) IsValid() bool {
	switch ae {
	case Development, Production:
		return true
	default:
		return false
	}
}
