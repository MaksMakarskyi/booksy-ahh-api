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

	return cfg, nil
}

type AppEnv string

const (
	Development AppEnv = "development"
	Production  AppEnv = "production"
)

var validAppEnvs = map[AppEnv]struct{}{
	Development: {},
	Production:  {},
}

func (ae AppEnv) IsValid() bool {
	_, ok := validAppEnvs[ae]
	return ok
}
