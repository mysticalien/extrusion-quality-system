package main

import (
	"context"
	authservice "extrusion-quality-system/internal/auth"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	httphandler "extrusion-quality-system/internal/http"
	"extrusion-quality-system/internal/ingestion"
	"extrusion-quality-system/internal/kafkaingest"
	"extrusion-quality-system/internal/mqttingest"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"strings"
	"time"
)

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
	fmt.Fprintln(w, `{"status":"ok"}`)
}

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

	logLevel.Set(parseLogLevel(cfg.Logging.Level))

	logger.Info(
		"configuration loaded",
		"serverAddr", cfg.Server.Addr,
		"databaseConfigured", cfg.Database.URL != "",
		"mqttEnabled", cfg.MQTT.Enabled,
		"kafkaEnabled", cfg.Kafka.Enabled,
		"mqttBrokerUrl", cfg.MQTT.BrokerURL,
		"kafkaBrokers", cfg.Kafka.BrokerList(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := storage.NewPostgresPool(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("connect to postgres failed", "error", err)
		os.Exit(1)
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

	authHandler := httphandler.NewAuthHandler(
		logger,
		userRepository,
		tokenManager,
	)

	userHandler := httphandler.NewUserHandler(
		logger,
		userRepository,
	)

	requireAuth := httphandler.AuthMiddleware(
		logger,
		tokenManager,
		userRepository,
	)

	protected := func(handler nethttp.HandlerFunc) nethttp.HandlerFunc {
		return requireAuth(handler).ServeHTTP
	}

	roles := func(handler nethttp.HandlerFunc, allowedRoles ...domain.UserRole) nethttp.HandlerFunc {
		return requireAuth(
			httphandler.RequireRoles(allowedRoles...)(
				handler,
			),
		).ServeHTTP
	}

	ingestionService := ingestion.NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
		ingestion.WithQualityWeightRepository(qualityWeightRepository),
	)

	var kafkaProducer *kafkaingest.Producer

	if cfg.Kafka.Enabled {
		kafkaProducer = kafkaingest.NewProducer(logger, cfg.Kafka)
		defer func() {
			if err := kafkaProducer.Close(); err != nil {
				logger.Error("close kafka producer failed", "error", err)
			}
		}()

		kafkaConsumer := kafkaingest.NewConsumer(
			logger,
			cfg.Kafka,
			ingestionService,
		)

		defer func() {
			if err := kafkaConsumer.Close(); err != nil {
				logger.Error("close kafka consumer failed", "error", err)
			}
		}()

		go func() {
			if err := kafkaConsumer.Start(context.Background()); err != nil {
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

	telemetryHandler := httphandler.NewTelemetryHandlerWithService(
		logger,
		ingestionService,
		telemetryRepository,
		setpointRepository,
	)

	setpointHandler := httphandler.NewSetpointHandler(logger, setpointRepository)

	eventHandler := httphandler.NewEventHandler(logger, alertRepository)
	qualityHandler := httphandler.NewQualityHandler(logger, qualityRepository)
	qualityWeightHandler := httphandler.NewQualityWeightHandler(logger, qualityWeightRepository)
	anomalyHandler := httphandler.NewAnomalyHandler(logger, anomalyRepository)

	logger.Info(
		"mqtt config loaded",
		"enabled", cfg.MQTT.Enabled,
		"brokerUrl", cfg.MQTT.BrokerURL,
		"topic", cfg.MQTT.TelemetryTopic,
		"qos", cfg.MQTT.QoS,
	)

	if cfg.MQTT.Enabled {
		if !cfg.Kafka.Enabled {
			logger.Error("mqtt ingestion requires kafka to be enabled")
			os.Exit(1)
		}

		mqttSubscriber := mqttingest.NewSubscriber(
			logger,
			cfg.MQTT,
			kafkaProducer,
		)

		go func() {
			if err := mqttSubscriber.Start(context.Background()); err != nil {
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

	// Operator+
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

	// Technologist+
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

	server := &nethttp.Server{
		Addr:              cfg.Server.Addr,
		Handler:           mux,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	logger.Info("server started", "addr", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
