package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MaksMakarskyi/booksy-go-api/internal/auth"
	"github.com/MaksMakarskyi/booksy-go-api/internal/hardware"
	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/rentals"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	_ "modernc.org/sqlite"
)

func NewServer(deps *dependencies.Registry) (*echo.Echo, error) {
	// Dependencies
	if deps == nil {
		return nil, fmt.Errorf("dependencies registry cannot be nil")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("dependencies registry config cannot be nil")
	}

	// Handlers
	hardwareHandler, err := hardware.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build hardware handler: %w", err)
	}

	rentalHandler, err := rentals.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build rental handler: %w", err)
	}

	profileHandler, err := profiles.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build profile handler: %w", err)
	}

	authHandler, err := auth.Build(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth handler: %w", err)
	}

	// Middleware
	authMiddleware, err := auth.Middleware(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth middleware: %w", err)
	}

	adminMiddleware := auth.AdminMiddleware()
	rateLimiterMiddlewre := middleware.RateLimiter(
		middleware.NewRateLimiterMemoryStore(deps.Config.RateLimitRPS),
	)

	// Server
	server := echo.New()
	server.Validator = deps.Validator
	server.JSONSerializer = deps.JSONSerializer
	server.HTTPErrorHandler = deps.ErrorHandler

	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.RequestID())
	server.Use(middleware.Recover())
	server.Use(middleware.CORSWithConfig(cors(deps.Config)))
	server.Use(middleware.ContextTimeout(time.Second * 15))
	server.Use(rateLimiterMiddlewre)

	server.GET("/healthz", healthz)

	hardwareGroup := server.Group("/hardware", authMiddleware)
	hardwareGroup.GET("", hardwareHandler.GetAll)
	hardwareGroup.POST("", hardwareHandler.Create, adminMiddleware)
	hardwareGroup.PATCH("/:id", hardwareHandler.Update, adminMiddleware)
	hardwareGroup.DELETE("/:id", hardwareHandler.Delete, adminMiddleware)
	hardwareGroup.PATCH("/:id/repair", hardwareHandler.MarkRepair, adminMiddleware)
	hardwareGroup.PATCH("/:id/available", hardwareHandler.MarkAvailable, adminMiddleware)
	hardwareGroup.GET("/search", hardwareHandler.Search)

	rentalGroup := server.Group("/rentals", authMiddleware)
	rentalGroup.GET("", rentalHandler.GetAll)
	rentalGroup.POST("", rentalHandler.Create)
	rentalGroup.PATCH("/:id/return", rentalHandler.Return)

	profileGroup := server.Group("/profiles", authMiddleware, adminMiddleware)
	profileGroup.GET("", profileHandler.GetAll)
	profileGroup.POST("", profileHandler.Create)
	profileGroup.DELETE("/:id", profileHandler.Delete)

	authGroup := server.Group("/auth")
	authGroup.POST("/token", authHandler.CreateToken)

	return server, nil
}

func healthz(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
}
