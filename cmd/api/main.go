package main

import (
	"context"
	"fmt"
	"log"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	deps, err := dependencies.NewRegistry(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to build dependecies: %v", err)
	}
	defer func() {
		if err := deps.Close(); err != nil {
			log.Fatalf("failed to close dependecies: %v", err)
		}
	}()

	created, err := profiles.EnsureAdmin(ctx, deps)
	if err != nil {
		log.Fatalf("failed to ensure admin profile: %v", err)
	}
	if created {
		log.Printf("created admin profile for %s", cfg.AdminEmail)
	} else {
		log.Printf("admin profile for %s already exists, left unchanged", cfg.AdminEmail)
	}

	server, err := server.NewServer(deps)
	if err != nil {
		log.Fatalf("failed to create a server: %v", err)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := server.Start(addr); err != nil {
		server.Logger.Error("failed to start server", "error", err)
	}
}
