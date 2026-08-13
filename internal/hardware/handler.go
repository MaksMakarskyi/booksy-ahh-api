package hardware

import (
	"fmt"
	"net/http"

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
		c.Logger().Error("failed to get hardwares from the store", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal error",
		})
	}

	return c.JSON(http.StatusOK, map[string][]Hardware{
		"data": hardwares,
	})
}
