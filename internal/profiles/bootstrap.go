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
	Email    string `json:"email"     validate:"required,max=255,email,endswith=@booksy.com"`
	Password string `json:"password"  validate:"required,min=8,maxbytes=72,password"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
}

// EnsureAdmins creates every account listed in ADMINS that does not exist yet
// and returns the emails it created. It is idempotent: an account already in
// the database is left untouched, including its password.
//
// The whole list is validated before anything is written, so a typo in the
// third entry cannot leave the first two created and the boot half-finished.
func EnsureAdmins(ctx context.Context, deps *dependencies.Registry) ([]string, error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies registry cannot be nil")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("dependencies registry config cannot be nil")
	}
	if deps.Validator == nil {
		return nil, fmt.Errorf("dependencies registry validator cannot be nil")
	}

	admins := make([]adminCredentials, 0, len(deps.Config.Admins))
	listed := make(map[string]int, len(deps.Config.Admins))

	for i, account := range deps.Config.Admins {
		position := i + 1

		credentials := adminCredentials{
			Email:    strings.ToLower(strings.TrimSpace(account.Email)),
			Password: account.Password,
			FullName: strings.TrimSpace(account.FullName),
		}

		if err := deps.Validator.Validate(&credentials); err != nil {
			return nil, fmt.Errorf("invalid admin at position %d: %w", position, err)
		}
		if first, duplicate := listed[credentials.Email]; duplicate {
			return nil, fmt.Errorf(
				"admin %s is listed twice, at positions %d and %d",
				credentials.Email, first, position,
			)
		}

		listed[credentials.Email] = position
		admins = append(admins, credentials)
	}

	if len(admins) == 0 {
		return nil, fmt.Errorf("at least one admin must be configured")
	}

	store, err := NewSQLiteStore(&SQLiteStoreOptions{Client: deps.DB})
	if err != nil {
		return nil, fmt.Errorf("failed to build store: %w", err)
	}

	created := make([]string, 0, len(admins))
	for _, credentials := range admins {
		passwordHash, err := pswutils.HashPassword(credentials.Password)
		if err != nil {
			return created, fmt.Errorf(
				"failed to hash password for admin %s: %w", credentials.Email, err,
			)
		}

		inserted, err := store.CreateAdminIfAbsent(ctx, NewProfile{
			Email:        credentials.Email,
			FullName:     credentials.FullName,
			Role:         roles.Admin,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return created, fmt.Errorf(
				"failed to ensure admin profile %s: %w", credentials.Email, err,
			)
		}

		if inserted {
			created = append(created, credentials.Email)
		}
	}

	return created, nil
}
