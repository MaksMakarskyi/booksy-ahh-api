package main

import (
	"context"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server"
)

func main() {
	ctx := context.Background()

	cfg, err := server.LoadConfig(ctx)
	if err != nil {
		panic(err)
	}

	server := server.NewServer()

	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := server.Start(addr); err != nil {
		server.Logger.Error("failed to start server", "error", err)
	}
}
