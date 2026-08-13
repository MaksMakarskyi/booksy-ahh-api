package validation

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v5"
)

type Normalizer interface {
	Normalize()
}

func DecodeJSON[T any](c *echo.Context) (T, error) {
	var v T

	if err := c.Bind(&v); err != nil {
		return v, echo.ErrBadRequest.Wrap(
			fmt.Errorf("failed to bind the request: %w", err),
		)
	}

	if normalizer, ok := any(&v).(Normalizer); ok {
		normalizer.Normalize()
	}

	if err := c.Validate(&v); err != nil {
		if validationErr, ok := errors.AsType[*ValidationError](err); ok {
			// Wrap the *ValidationError itself rather than its message: the
			// central error handler unwraps it to render per-field details.
			// Formatting it into a string here would discard them.
			return v, echo.ErrBadRequest.Wrap(validationErr)
		}

		return v, echo.ErrInternalServerError.Wrap(
			fmt.Errorf("failed to validate the request: %w", err),
		)
	}

	return v, nil
}
