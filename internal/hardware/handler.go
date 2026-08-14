package hardware

import (
	"fmt"
	"net/http"
	"strconv"

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
		return err
	}

	return c.JSON(http.StatusOK, map[string][]Hardware{
		"data": hardwares,
	})
}

func (h *Handler) Create(c *echo.Context) error {
	var newHardware NewHardware
	if err := valutils.DecodeJSON(c, &newHardware); err != nil {
		return err
	}

	storedHardware, err := h.store.Create(c.Request().Context(), newHardware)
	if err != nil {
		return fmt.Errorf("failed to store harware: %w", err)
	}

	return c.JSON(http.StatusCreated, map[string]Hardware{
		"data": storedHardware,
	})
}

func (h *Handler) Update(c *echo.Context) error {
	var updatedHardware UpdatedHardware
	if err := valutils.DecodeJSON(c, &updatedHardware); err != nil {
		return err
	}

	storedHardware, err := h.store.Update(c.Request().Context(), updatedHardware)
	if err != nil {
		return fmt.Errorf("failed to update harware %d: %w", updatedHardware.ID, err)
	}

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": storedHardware,
	})
}

func (h *Handler) Delete(c *echo.Context) error {
	hardwareID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid 'id' path parameter: %s", c.Param("id"))
	}

	deletedHardware, err := h.store.Delete(c.Request().Context(), hardwareID)
	if err != nil {
		return fmt.Errorf("failed to delete harware %d: %w", hardwareID, err)
	}

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": deletedHardware,
	})
}
