package validation

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/labstack/echo/v5"
)

type StrictJSONSerializer struct {
	echo.DefaultJSONSerializer
}

func (StrictJSONSerializer) Deserialize(c *echo.Context, target any) error {
	dec := json.NewDecoder(c.Request().Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(target); err != nil {
		return echo.ErrBadRequest.Wrap(unknownFieldError(err))
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return echo.ErrBadRequest.Wrap(
			fmt.Errorf("request body must contain a single JSON value"),
		)
	}

	return nil
}

func unknownFieldError(err error) error {
	const prefix = "json: unknown field "

	msg := err.Error()
	if !strings.HasPrefix(msg, prefix) {
		return err
	}

	field := strings.Trim(strings.TrimPrefix(msg, prefix), `"`)

	return fmt.Errorf("unknown field %q is not accepted by this endpoint", field)
}
