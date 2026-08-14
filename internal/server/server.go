package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MaksMakarskyi/booksy-go-api/internal/hardware"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	_ "modernc.org/sqlite"
)

func NewServer(deps *dependencies.Registry) (*echo.Echo, error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies registry cannot be nil")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("dependencies registry config cannot be nil")
	}

	hardwareHandler, err := hardware.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build hardware handler: %w", err)
	}

	server := echo.New()
	server.Validator = deps.Validator
	server.JSONSerializer = deps.JSONSerializer
	server.HTTPErrorHandler = deps.ErrorHandler

	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.RequestID())
	server.Use(middleware.Recover())
	server.Use(middleware.CORSWithConfig(cors(deps.Config)))
	server.Use(middleware.ContextTimeout(time.Second * 15))

	server.GET("/healthz", healthz)

	hardwareGroup := server.Group("/hardware")
	hardwareGroup.GET("", hardwareHandler.GetAll)
	hardwareGroup.POST("", hardwareHandler.Create)       // TODO: cover with admin middleware
	hardwareGroup.PATCH("/:id", hardwareHandler.Update)  // TODO: cover with admin middleware
	hardwareGroup.DELETE("/:id", hardwareHandler.Delete) // TODO: cover with admin middleware

	return server, nil
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
}
