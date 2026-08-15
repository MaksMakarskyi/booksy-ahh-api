package profiles

import (
	"context"
	"fmt"
	"strings"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles/roles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	pswutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/password"
)

type adminCredentials struct {
	Email    string `json:"ADMIN_EMAIL"    validate:"required,max=255,email"`
	Password string `json:"ADMIN_PASSWORD" validate:"required,min=8,maxbytes=72,password"`
	FullName string `json:"ADMIN_NAME"     validate:"required,min=2,max=255"`
}

func EnsureAdmin(ctx context.Context, deps *dependencies.Registry) (bool, error) {
	if deps == nil {
		return false, fmt.Errorf("dependencies registry cannot be nil")
	}
	if deps.Config == nil {
		return false, fmt.Errorf("dependencies registry config cannot be nil")
	}
	if deps.Validator == nil {
		return false, fmt.Errorf("dependencies registry validator cannot be nil")
	}

	credentials := adminCredentials{
		Email:    strings.ToLower(strings.TrimSpace(deps.Config.AdminEmail)),
		Password: deps.Config.AdminPassword,
		FullName: strings.TrimSpace(deps.Config.AdminName),
	}

	if err := deps.Validator.Validate(&credentials); err != nil {
		return false, fmt.Errorf("invalid admin credentials: %w", err)
	}

	store, err := NewSQLiteStore(&SQLiteStoreOptions{Client: deps.DB})
	if err != nil {
		return false, fmt.Errorf("failed to build store: %w", err)
	}

	passwordHash, err := pswutils.HashPassword(credentials.Password)
	if err != nil {
		return false, fmt.Errorf("failed to hash admin password: %w", err)
	}

	created, err := store.CreateAdminIfAbsent(ctx, NewProfile{
		Email:        credentials.Email,
		FullName:     credentials.FullName,
		Role:         roles.Admin,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return false, fmt.Errorf("failed to ensure admin profile: %w", err)
	}

	return created, nil
}
