package profiles

import (
	"fmt"
	"net/http"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	pswutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/password"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type Handler struct {
	store Store
}

type HandlerOptions struct {
	Store Store
}

func NewHandler(opts *HandlerOptions) (*Handler, error) {
	if opts == nil {
		return nil, fmt.Errorf("HandlerOptions cannot be nil")
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("HandlerOptions.Store cannot be nil")
	}

	handler := &Handler{
		store: opts.Store,
	}

	return handler, nil
}

func (h *Handler) Create(c *echo.Context) error {
	var req CreateProfileReq
	if err := valutils.DecodeJSON(c, &req); err != nil {
		return err
	}

	passwordHash, err := pswutils.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("%w: %w", errutils.ErrGeneratePasswordHash, err)
	}

	storedProfile, err := h.store.Create(c.Request().Context(), NewProfile{
		Email:        req.Email,
		FullName:     req.FullName,
		Role:         req.Role,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("failed to store profile: %w", err)
	}

	return c.JSON(http.StatusCreated, map[string]Profile{
		"data": storedProfile,
	})
}
