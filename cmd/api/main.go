package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	created, err := profiles.EnsureAdmins(ctx, deps)
	if err != nil {
		log.Fatalf("failed to ensure admin profiles: %v", err)
	}
	if len(created) > 0 {
		log.Printf("created %d admin profile(s): %s", len(created), strings.Join(created, ", "))
	} else {
		log.Printf("all %d admin profile(s) already exist, left unchanged", len(cfg.Admins))
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
