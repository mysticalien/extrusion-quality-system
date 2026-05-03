package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"extrusion-quality-system/internal/adapters/postgres"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/ports"
	"extrusion-quality-system/internal/security/password"
	"extrusion-quality-system/internal/security/token"
	authusecase "extrusion-quality-system/internal/usecase/auth"
	"extrusion-quality-system/internal/usecase/telemetry"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dependencies struct {
	pool *pgxpool.Pool

	telemetryRepository     ports.TelemetryRepository
	alertRepository         ports.AlertRepository
	qualityRepository       ports.QualityRepository
	setpointRepository      ports.SetpointRepository
	anomalyRepository       ports.AnomalyRepository
	qualityWeightRepository ports.QualityWeightRepository
	userRepository          ports.UserRepository

	tokenManager   ports.TokenManager
	passwordHasher ports.PasswordHasher

	authService      *authusecase.Service
	telemetryService *telemetry.Service
}

func buildDependencies(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
) (*dependencies, func(), error) {
	startupCtx, cancelStartup := context.WithTimeout(ctx, 5*time.Second)
	defer cancelStartup()

	pool, err := postgres.NewPool(startupCtx, cfg.Database.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}

	cleanup := func() {
		pool.Close()
	}

	logger.Info("database connected", "databaseConfigured", cfg.Database.URL != "")

	telemetryRepository := postgres.NewTelemetryRepository(pool)
	alertRepository := postgres.NewAlertRepository(pool)
	qualityRepository := postgres.NewQualityRepository(pool)
	setpointRepository := postgres.NewSetpointRepository(pool)
	anomalyRepository := postgres.NewAnomalyRepository(pool)
	qualityWeightRepository := postgres.NewQualityWeightRepository(pool)
	userRepository := postgres.NewUserRepository(pool)

	passwordHasher := password.NewBcryptHasher(cfg.Auth.BcryptCost)

	tokenManager, err := token.NewJWTManager(
		cfg.Auth.TokenSecret,
		cfg.Auth.TokenTTL,
		cfg.Auth.TokenIssuer,
	)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("create jwt manager: %w", err)
	}

	authService := authusecase.NewService(
		userRepository,
		passwordHasher,
		tokenManager,
	)

	telemetryService := telemetry.NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
		telemetry.WithQualityWeightRepository(qualityWeightRepository),
	)

	return &dependencies{
		pool: pool,

		telemetryRepository:     telemetryRepository,
		alertRepository:         alertRepository,
		qualityRepository:       qualityRepository,
		setpointRepository:      setpointRepository,
		anomalyRepository:       anomalyRepository,
		qualityWeightRepository: qualityWeightRepository,
		userRepository:          userRepository,

		tokenManager:   tokenManager,
		passwordHasher: passwordHasher,

		authService:      authService,
		telemetryService: telemetryService,
	}, cleanup, nil
}
