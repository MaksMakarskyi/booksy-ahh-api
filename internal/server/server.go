package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func NewServer() *echo.Echo {
	server := echo.New()

	server.GET("/healz", healz)

	return server
}

func healz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
}
