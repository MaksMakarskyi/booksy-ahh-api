package auth

import (
	"context"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type authKey string

const userKey authKey = "user"

func Middleware(deps *dependencies.Registry) (echo.MiddlewareFunc, error) {
	if deps == nil {
		return nil, fmt.Errorf("deps cannot be nil")
	}
	if deps.JWTIssuer == nil {
		return nil, fmt.Errorf("token issuer cannot be nil")
	}

	issuer := deps.JWTIssuer

	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:" + echo.HeaderAuthorization + ":Bearer ",
		Validator: func(c *echo.Context, key string, _ middleware.ExtractorSource) (bool, error) {
			claims, err := issuer.Verify(key)
			if err != nil {
				return false, fmt.Errorf("%w: %w", errutils.ErrInvalidBearerToken, err)
			}

			userID, err := claims.UserID()
			if err != nil {
				return false, fmt.Errorf("%w: %w", errutils.ErrInvalidBearerToken, err)
			}

			role := profiles.ProfileRole(claims.Role)
			if !role.IsValid() {
				return false, fmt.Errorf(
					"%w: unknown role %q", errutils.ErrInvalidBearerToken, claims.Role,
				)
			}

			user := &User{
				ID:       userID,
				Email:    claims.Email,
				FullName: claims.FullName,
				Role:     role,
			}

			c.SetRequest(c.Request().WithContext(
				context.WithValue(c.Request().Context(), userKey, user),
			))

			return true, nil
		},
	}), nil
}

func AdminMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, err := GetUser(c)
			if err != nil {
				return err
			}

			if user.Role != profiles.Admin {
				return echo.ErrForbidden
			}

			return next(c)
		}
	}
}

func GetUser(c *echo.Context) (*User, error) {
	user, ok := c.Request().Context().Value(userKey).(*User)
	if !ok || user == nil {
		return nil, errutils.ErrUserNotFound
	}

	return user, nil
}
