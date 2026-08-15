package config

import (
	"context"
	"encoding/json"
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

	RateLimitRPS float64 `env:"RATE_LIMIT_RPS, default=15"`

	GooseTable string `env:"GOOSE_TABLE, default=goose_migrations"`

	Admins AdminAccounts `env:"ADMINS, required"`
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
	if cfg.RateLimitRPS <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT_RPS must be positive, got %v", cfg.RateLimitRPS)
	}
	if len(cfg.Admins) == 0 {
		return nil, fmt.Errorf("ADMINS must list at least one account")
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

type AdminAccount struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type AdminAccounts []AdminAccount

var _ envconfig.Decoder = (*AdminAccounts)(nil)

func (aa *AdminAccounts) EnvDecode(val string) error {
	decoder := json.NewDecoder(strings.NewReader(val))
	decoder.DisallowUnknownFields()

	var accounts []AdminAccount
	if err := decoder.Decode(&accounts); err != nil {
		return fmt.Errorf(
			`ADMINS must be a JSON array of {"email","full_name","password"} objects: %w`, err,
		)
	}

	*aa = accounts

	return nil
}
