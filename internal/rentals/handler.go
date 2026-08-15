package rentals

import (
	"fmt"
	"net/http"

	"github.com/MaksMakarskyi/booksy-go-api/internal/auth"
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

	return &Handler{store: opts.Store}, nil
}

func (h *Handler) GetAll(c *echo.Context) error {
	user, err := auth.GetUser(c)
	if err != nil {
		return err
	}

	rentals, err := h.store.GetAll(c.Request().Context(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to list rentals for user %d: %w", user.ID, err)
	}

	return c.JSON(http.StatusOK, map[string][]RentalWithHardware{
		"data": rentals,
	})
}

func (h *Handler) Create(c *echo.Context) error {
	user, err := auth.GetUser(c)
	if err != nil {
		return err
	}

	var req CreateRentalReq
	if err := valutils.DecodeJSON(c, &req); err != nil {
		return err
	}

	rental, err := h.store.Create(c.Request().Context(), req.HardwareID, user.ID)
	if err != nil {
		return fmt.Errorf(
			"failed to rent hardware %d for user %d: %w", req.HardwareID, user.ID, err,
		)
	}

	return c.JSON(http.StatusCreated, map[string]Rental{
		"data": rental,
	})
}

func (h *Handler) Return(c *echo.Context) error {
	user, err := auth.GetUser(c)
	if err != nil {
		return err
	}

	rentalID, err := valutils.PathInt(c, "id")
	if err != nil {
		return err
	}

	rental, err := h.store.Return(c.Request().Context(), rentalID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to return rental %d: %w", rentalID, err)
	}

	return c.JSON(http.StatusOK, map[string]Rental{
		"data": rental,
	})
}
