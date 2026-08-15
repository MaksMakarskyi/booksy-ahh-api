package main

import (
	"context"
	"fmt"
	"log"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	"github.com/MaksMakarskyi/booksy-go-api/internal/utils/migrate"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	deps, err := dependencies.NewRegistry(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to build dependencies: %v", err)
	}
	defer func() {
		if err := deps.Close(); err != nil {
			log.Printf("failed to close dependencies: %v", err)
		}
	}()

	if err := migrate.Up(deps.DB, cfg.GooseTable); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	created, err := profiles.EnsureAdmin(ctx, deps)
	if err != nil {
		log.Fatalf("failed to ensure admin profile: %v", err)
	}
	if created {
		log.Print("created admin profile")
	} else {
		log.Print("admin profile already exists, left unchanged")
	}

	srv, err := server.NewServer(deps)
	if err != nil {
		log.Fatalf("failed to create a server: %v", err)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := srv.Start(addr); err != nil {
		srv.Logger.Error("failed to start server", "error", err)
	}
}
