package server

import (
	httpadapter "extrusion-quality-system/internal/adapters/http"
	"extrusion-quality-system/internal/domain"
	"log/slog"
	nethttp "net/http"
)

func newRouter(
	logger *slog.Logger,
	deps *dependencies,
) nethttp.Handler {
	authHandler := httpadapter.NewAuthHandler(
		logger,
		deps.authService,
		deps.tokenManager,
	)

	userHandler := httpadapter.NewUserHandler(
		logger,
		deps.userRepository,
		deps.passwordHasher,
	)

	requireAuth := httpadapter.AuthMiddleware(
		logger,
		deps.tokenManager,
		deps.userRepository,
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
		deps.telemetryService,
		deps.telemetryRepository,
		deps.setpointRepository,
	)

	setpointHandler := httpadapter.NewSetpointHandler(logger, deps.setpointRepository)
	eventHandler := httpadapter.NewEventHandler(logger, deps.alertRepository)
	qualityHandler := httpadapter.NewQualityHandler(logger, deps.qualityRepository)
	qualityWeightHandler := httpadapter.NewQualityWeightHandler(logger, deps.qualityWeightRepository)
	anomalyHandler := httpadapter.NewAnomalyHandler(logger, deps.anomalyRepository)

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
