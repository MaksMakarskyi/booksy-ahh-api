package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	jwtutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/jwt"
	pswutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/password"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type Handler struct {
	store     Store
	jwtIssuer *jwtutils.Issuer
}

type HandlerOptions struct {
	Store     Store
	JWTIssuer *jwtutils.Issuer
}

func NewHandler(opts *HandlerOptions) (*Handler, error) {
	if opts == nil {
		return nil, fmt.Errorf("HandlerOptions cannot be nil")
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("HandlerOptions.Store cannot be nil")
	}
	if opts.JWTIssuer == nil {
		return nil, fmt.Errorf("HandlerOptions.Issuer cannot be nil")
	}

	handler := &Handler{
		store:     opts.Store,
		jwtIssuer: opts.JWTIssuer,
	}

	return handler, nil
}

func (h *Handler) CreateToken(c *echo.Context) error {
	var req CreateTokenReq
	if err := valutils.DecodeJSON(c, &req); err != nil {
		return err
	}

	credentials, err := h.store.GetUserWithCreds(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, errutils.ErrStoreNotFound) {
			return errutils.ErrInvalidCredentials
		}

		return fmt.Errorf("failed to get credentials: %w", err)
	}

	if !pswutils.CheckPassword(credentials.PasswordHash, req.Password) {
		return errutils.ErrInvalidCredentials
	}

	token, expiresAt, err := h.jwtIssuer.Issue(jwtutils.Identity{
		ID:       credentials.ID,
		Email:    credentials.Email,
		FullName: credentials.FullName,
		Role:     string(credentials.Role),
	})
	if err != nil {
		return fmt.Errorf("failed to issue token: %w", err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": map[string]any{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_at":   expiresAt.Format(time.RFC3339),
		},
	})
}
