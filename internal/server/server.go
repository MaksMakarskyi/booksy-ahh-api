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
	hardwareHandler, err := hardware.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build hardware handler: %w", err)
	}

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORSWithConfig(cors(deps.Config)))
	server.Use(middleware.ContextTimeout(time.Second * 15))

	server.GET("/healthz", healthz)

	hardwareGroup := server.Group("/hardware")
	hardwareGroup.GET("", hardwareHandler.GetAll)

	return server, nil
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
}
