package errors

import (
	"context"
	"errors"
	"net/http"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message   string                `json:"message"`
	Fields    []valutils.FieldError `json:"fields,omitempty"`
	RequestID string                `json:"request_id,omitempty"`
	Debug     string                `json:"debug,omitempty"`
}

func NewErrorHandler(cfg *config.Config) (echo.HTTPErrorHandler, error) {
	if cfg == nil {
		return nil, errors.New("config.Config cannot be nil")
	}

	isDev := cfg.Env == config.Development

	return func(c *echo.Context, err error) {
		if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Committed {
			return
		}

		status, body := translate(err)

		if errors.Is(err, context.Canceled) {
			c.Logger().Debug("client cancelled the request", "path", c.Request().URL.Path)
			return
		}

		if id := requestID(c); id != "" {
			body.RequestID = id
		}
		if isDev {
			body.Debug = err.Error()
		}

		logAttrs := []any{
			"status", status,
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"request_id", body.RequestID,
			"error", err,
		}
		if status >= http.StatusInternalServerError {
			c.Logger().Error("request failed", logAttrs...)
		} else {
			c.Logger().Info("request rejected", logAttrs...)
		}

		var writeErr error
		if c.Request().Method == http.MethodHead {
			writeErr = c.NoContent(status)
		} else {
			writeErr = c.JSON(status, ErrorResponse{Error: body})
		}

		if writeErr != nil {
			c.Logger().Error("failed to write error response",
				"error", errors.Join(err, writeErr))
		}
	}, nil
}

// translate maps an error onto a status code and a client-safe body.
func translate(err error) (int, ErrorBody) {
	if validationErr, ok := errors.AsType[*valutils.ValidationError](err); ok {
		return http.StatusBadRequest, ErrorBody{
			Message: "The request payload is invalid.",
			Fields:  validationErr.Fields,
		}
	}

	switch {
	case errors.Is(err, ErrStoreNotFound):
		return http.StatusNotFound, ErrorBody{Message: "The requested resource does not exist."}
	case errors.Is(err, ErrInvalidBearerToken):
		return http.StatusUnauthorized, ErrorBody{Message: "Missing or invalid access token."}
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, ErrorBody{Message: "Invalid email or password."}
	case errors.Is(err, ErrStoreForbidden):
		return http.StatusForbidden, ErrorBody{Message: "You are not allowed to perform this action."}
	case errors.Is(err, ErrStoreConflict):
		return http.StatusConflict, ErrorBody{Message: "The request conflicts with the current state of the resource."}
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, ErrorBody{Message: "The request took too long. Please try again."}
	}

	if httpErr, ok := errors.AsType[*echo.HTTPError](err); ok {
		return httpErr.Code, ErrorBody{Message: safeMessage(httpErr)}
	}

	var coder echo.HTTPStatusCoder
	if errors.As(err, &coder) {
		if code := coder.StatusCode(); code != 0 {
			return code, ErrorBody{Message: http.StatusText(code)}
		}
	}

	return http.StatusInternalServerError, ErrorBody{Message: http.StatusText(http.StatusInternalServerError)}
}

func safeMessage(httpErr *echo.HTTPError) string {
	if httpErr.Message != "" && httpErr.Code < http.StatusInternalServerError {
		return httpErr.Message
	}

	return http.StatusText(httpErr.Code)
}

func requestID(c *echo.Context) string {
	if id := c.Response().Header().Get(echo.HeaderXRequestID); id != "" {
		return id
	}

	return c.Request().Header.Get(echo.HeaderXRequestID)
}
