package hardware

import (
	"fmt"
	"net/http"

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

func (h *Handler) GetAll(c *echo.Context) error {
	hardwares, err := h.store.GetAll(c.Request().Context())
	if err != nil {
		return echo.ErrInternalServerError.Wrap(
			fmt.Errorf("failed to get hardwares from the store: %w", err),
		)
	}

	return c.JSON(http.StatusOK, map[string][]Hardware{
		"data": hardwares,
	})
}

func (h *Handler) Create(c *echo.Context) error {
	newHardware, err := valutils.DecodeJSON[NewHardware](c)
	if err != nil {
		return err
	}

	storedHardware, err := h.store.Create(c.Request().Context(), newHardware)
	if err != nil {
		return echo.ErrInternalServerError.Wrap(
			fmt.Errorf("failed to store new hardware: %w", err),
		)
	}

	return c.JSON(http.StatusCreated, map[string]Hardware{
		"data": storedHardware,
	})
}
