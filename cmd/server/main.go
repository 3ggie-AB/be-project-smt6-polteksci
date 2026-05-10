package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"project_smt6/internal/app"
	"project_smt6/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := config.NewLogger(cfg.Server.Environment)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("application failed", "error", err)
		log.Fatal(err)
	}
}
