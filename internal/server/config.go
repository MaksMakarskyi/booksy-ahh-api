package server

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Env  AppEnv `env:"APP_ENV, default=production"`
	Port string `env:"PORT, default=8080"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := new(Config)

	if err := envconfig.Process(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to load env vars: %w", err)
	}

	if !cfg.Env.IsValid() {
		return nil, fmt.Errorf("APP_ENV is not valid, got: %q", cfg.Env)
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
