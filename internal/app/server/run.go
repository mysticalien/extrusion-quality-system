package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"time"

	"extrusion-quality-system/internal/config"
)

const shutdownTimeout = 10 * time.Second

func Run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	logger.Info(
		"configuration loaded",
		"serverAddr", cfg.Server.Addr,
		"databaseConfigured", cfg.Database.URL != "",
		"mqttEnabled", cfg.MQTT.Enabled,
		"kafkaEnabled", cfg.Kafka.Enabled,
		"mqttBrokerUrl", cfg.MQTT.BrokerURL,
		"kafkaBrokers", cfg.Kafka.BrokerList(),
	)

	dependencies, cleanup, err := buildDependencies(ctx, logger, cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := startTelemetryPipeline(
		ctx,
		logger,
		cfg,
		dependencies.telemetryService,
	); err != nil {
		return err
	}

	httpServer := &nethttp.Server{
		Addr:              cfg.Server.Addr,
		Handler:           newRouter(logger, dependencies),
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("server started", "addr", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("server shutdown requested")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		logger.Info("server stopped")
		return nil

	case err := <-errCh:
		return err
	}
}
