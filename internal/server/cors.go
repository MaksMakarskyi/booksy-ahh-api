package server

import (
	"net/http"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	"github.com/labstack/echo/v5/middleware"
)

func cors(cfg *config.Config) middleware.CORSConfig {
	var corsCfg middleware.CORSConfig

	switch cfg.Env {
	case config.Development:
		corsCfg = middleware.CORSConfig{
			AllowMethods: []string{"*"},
			AllowHeaders: []string{"*"},
			AllowOrigins: []string{"*"},
		}
	case config.Production:
		corsCfg = middleware.CORSConfig{
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
			},
			AllowHeaders:     []string{"*"},
			AllowCredentials: true,
			AllowOrigins:     []string{"https://myapp.vercel.app"},
		}
	}

	return corsCfg
}
