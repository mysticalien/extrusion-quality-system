package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"extrusion-quality-system/internal/app/logging"
	simulatorapp "extrusion-quality-system/internal/app/simulator"
	"extrusion-quality-system/internal/config"
)

func main() {
	var logLevel slog.LevelVar
	logLevel.Set(slog.LevelInfo)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: &logLevel,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	logLevel.Set(logging.ParseLevel(cfg.Logging.Level))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := simulatorapp.Run(ctx, logger, cfg.Simulator); err != nil {
		logger.Error("simulator stopped with error", "error", err)
		os.Exit(1)
	}
}
