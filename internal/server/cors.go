package server

import (
	"net/http"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	"github.com/labstack/echo/v5/middleware"
)

func cors(cfg *config.Config) middleware.CORSConfig {
	if cfg.Env == config.Development {
		return middleware.CORSConfig{
			AllowMethods: []string{"*"},
			AllowHeaders: []string{"*"},
			AllowOrigins: []string{"*"},
		}
	}

	return middleware.CORSConfig{
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowHeaders: []string{"*"},
		AllowOrigins: cfg.CORSOrigins,
	}
}
