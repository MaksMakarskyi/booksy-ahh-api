package validation

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/labstack/echo/v5"
)

// StrictJSONSerializer rejects request bodies containing fields the target
// struct does not declare.
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

// unknownFieldError rewrites encoding/json's message into something a client
// can act on. The stdlib phrasing is:
//
//	json: unknown field "stauts"
func unknownFieldError(err error) error {
	const prefix = "json: unknown field "

	msg := err.Error()
	if !strings.HasPrefix(msg, prefix) {
		return err
	}

	field := strings.Trim(strings.TrimPrefix(msg, prefix), `"`)

	return fmt.Errorf("unknown field %q is not accepted by this endpoint", field)
}
