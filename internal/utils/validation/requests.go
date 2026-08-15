package validation

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/labstack/echo/v5"
)

type Normalizer interface {
	Normalize()
}

func DecodeJSON[T any](c *echo.Context, v *T) error {
	if err := c.Bind(v); err != nil {
		return echo.ErrBadRequest.Wrap(
			fmt.Errorf("failed to bind the request: %w", err),
		)
	}

	if normalizer, ok := any(v).(Normalizer); ok {
		normalizer.Normalize()
	}

	if err := c.Validate(v); err != nil {
		if validationErr, ok := errors.AsType[*ValidationError](err); ok {
			return echo.ErrBadRequest.Wrap(validationErr)
		}

		return echo.ErrInternalServerError.Wrap(
			fmt.Errorf("failed to validate the request: %w", err),
		)
	}

	return nil
}

func PathInt(c *echo.Context, name string) (int, error) {
	raw := c.Param(name)

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, echo.ErrBadRequest.Wrap(
			fmt.Errorf("invalid %q path parameter: %s", name, raw),
		)
	}

	return value, nil
}
