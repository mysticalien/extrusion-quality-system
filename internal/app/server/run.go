package server

import (
	"context"
	"errors"
	httpadapter "extrusion-quality-system/internal/adapters/http"
	"extrusion-quality-system/internal/adapters/kafka"
	mqttadapter "extrusion-quality-system/internal/adapters/mqtt"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"time"

	authservice "extrusion-quality-system/internal/auth"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"extrusion-quality-system/internal/storage"
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

	startupCtx, cancelStartup := context.WithTimeout(ctx, 5*time.Second)
	defer cancelStartup()

	pool, err := storage.NewPostgresPool(startupCtx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	logger.Info("database connected", "databaseConfigured", cfg.Database.URL != "")

	telemetryRepository := storage.NewPostgresTelemetryRepository(pool)
	alertRepository := storage.NewPostgresAlertRepository(pool)
	qualityRepository := storage.NewPostgresQualityRepository(pool)
	setpointRepository := storage.NewPostgresSetpointRepository(pool)
	anomalyRepository := storage.NewPostgresAnomalyRepository(pool)
	qualityWeightRepository := storage.NewPostgresQualityWeightRepository(pool)
	userRepository := storage.NewPostgresUserRepository(pool)

	tokenManager := authservice.NewTokenManager(
		cfg.Auth.TokenSecret,
		cfg.Auth.TokenTTL,
	)

	ingestionService := ingestion.NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
		ingestion.WithQualityWeightRepository(qualityWeightRepository),
	)

	if err := startTelemetryIngestion(ctx, logger, cfg, ingestionService); err != nil {
		return err
	}

	mux := newRouter(
		logger,
		tokenManager,
		userRepository,
		ingestionService,
		telemetryRepository,
		setpointRepository,
		alertRepository,
		qualityRepository,
		qualityWeightRepository,
		anomalyRepository,
	)

	httpServer := &nethttp.Server{
		Addr:              cfg.Server.Addr,
		Handler:           mux,
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

func startTelemetryIngestion(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	ingestionService *ingestion.Service,
) error {
	var kafkaProducer *kafkaadapter.Producer

	if cfg.Kafka.Enabled {
		kafkaProducer = kafkaadapter.NewProducer(logger, cfg.Kafka)

		kafkaConsumer := kafkaadapter.NewConsumer(
			logger,
			cfg.Kafka,
			ingestionService,
		)

		go func() {
			<-ctx.Done()

			if err := kafkaProducer.Close(); err != nil {
				logger.Error("close kafka producer failed", "error", err)
			}

			if err := kafkaConsumer.Close(); err != nil {
				logger.Error("close kafka consumer failed", "error", err)
			}
		}()

		go func() {
			if err := kafkaConsumer.Start(ctx); err != nil {
				logger.Error("kafka consumer stopped with error", "error", err)
			}
		}()

		logger.Info(
			"kafka ingestion enabled",
			"brokers", cfg.Kafka.BrokerList(),
			"topic", cfg.Kafka.TelemetryTopic,
			"consumerGroup", cfg.Kafka.ConsumerGroup,
		)
	}

	logger.Info(
		"mqtt config loaded",
		"enabled", cfg.MQTT.Enabled,
		"brokerUrl", cfg.MQTT.BrokerURL,
		"topic", cfg.MQTT.TelemetryTopic,
		"qos", cfg.MQTT.QoS,
	)

	if cfg.MQTT.Enabled {
		if !cfg.Kafka.Enabled {
			return errors.New("mqtt ingestion requires kafka to be enabled")
		}

		if kafkaProducer == nil {
			return errors.New("kafka producer is not configured")
		}

		mqttSubscriber := mqttadapter.NewSubscriber(
			logger,
			cfg.MQTT,
			kafkaProducer,
		)

		go func() {
			if err := mqttSubscriber.Start(ctx); err != nil {
				logger.Error("mqtt subscriber stopped with error", "error", err)
			}
		}()

		logger.Info(
			"mqtt to kafka bridge enabled",
			"mqttBrokerUrl", cfg.MQTT.BrokerURL,
			"mqttTopic", cfg.MQTT.TelemetryTopic,
			"kafkaTopic", cfg.Kafka.TelemetryTopic,
		)
	}

	return nil
}

func newRouter(
	logger *slog.Logger,
	tokenManager *authservice.TokenManager,
	userRepository storage.UserRepository,
	ingestionService *ingestion.Service,
	telemetryRepository storage.TelemetryRepository,
	setpointRepository storage.SetpointRepository,
	alertRepository storage.AlertRepository,
	qualityRepository storage.QualityRepository,
	qualityWeightRepository storage.QualityWeightRepository,
	anomalyRepository storage.AnomalyRepository,
) nethttp.Handler {
	authHandler := httpadapter.NewAuthHandler(
		logger,
		userRepository,
		tokenManager,
	)

	userHandler := httpadapter.NewUserHandler(
		logger,
		userRepository,
	)

	requireAuth := httpadapter.AuthMiddleware(
		logger,
		tokenManager,
		userRepository,
	)

	protected := func(handler nethttp.HandlerFunc) nethttp.HandlerFunc {
		return requireAuth(handler).ServeHTTP
	}

	roles := func(handler nethttp.HandlerFunc, allowedRoles ...domain.UserRole) nethttp.HandlerFunc {
		return requireAuth(
			httpadapter.RequireRoles(allowedRoles...)(
				handler,
			),
		).ServeHTTP
	}

	telemetryHandler := httpadapter.NewTelemetryHandlerWithService(
		logger,
		ingestionService,
		telemetryRepository,
		setpointRepository,
	)

	setpointHandler := httpadapter.NewSetpointHandler(logger, setpointRepository)
	eventHandler := httpadapter.NewEventHandler(logger, alertRepository)
	qualityHandler := httpadapter.NewQualityHandler(logger, qualityRepository)
	qualityWeightHandler := httpadapter.NewQualityWeightHandler(logger, qualityWeightRepository)
	anomalyHandler := httpadapter.NewAnomalyHandler(logger, anomalyRepository)

	mux := nethttp.NewServeMux()

	mux.Handle("/static/", nethttp.StripPrefix("/static/", nethttp.FileServer(nethttp.Dir("web"))))

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)

	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/me", protected(authHandler.Me))
	mux.HandleFunc("/api/me/change-password", protected(authHandler.ChangePassword))

	mux.HandleFunc("/api/users", roles(
		userHandler.ListCreate,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/users/", roles(
		userHandler.Action,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/telemetry", protected(telemetryHandler.Create))

	mux.HandleFunc("/api/telemetry/latest", roles(
		telemetryHandler.Latest,
		domain.UserRoleOperator,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/events", roles(
		eventHandler.List,
		domain.UserRoleOperator,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/events/active", roles(
		eventHandler.Active,
		domain.UserRoleOperator,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/events/", roles(
		eventHandler.Action,
		domain.UserRoleOperator,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/quality/latest", roles(
		qualityHandler.Latest,
		domain.UserRoleOperator,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/quality/weights", roles(
		qualityWeightHandler.List,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/quality/weights/", roles(
		qualityWeightHandler.Update,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/telemetry/history", roles(
		telemetryHandler.History,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/quality/history", roles(
		qualityHandler.History,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/anomalies", roles(
		anomalyHandler.List,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/anomalies/active", roles(
		anomalyHandler.Active,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/setpoints", roles(
		setpointHandler.List,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	mux.HandleFunc("/api/setpoints/", roles(
		setpointHandler.Update,
		domain.UserRoleTechnologist,
		domain.UserRoleAdmin,
	))

	return mux
}

func homeHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	nethttp.ServeFile(w, r, "web/index.html")
}

func healthHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
}
