package main

import (
	"context"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	httphandler "extrusion-quality-system/internal/http"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"time"
)

func homeHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if r.URL.Path != "/" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	fmt.Fprint(w, "Homepage!")
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := storage.NewPostgresPool(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("connect to postgres failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("connected to postgres")

	telemetryRepository := storage.NewPostgresTelemetryRepository(pool)
	alertRepository := storage.NewPostgresAlertRepository(pool)
	qualityRepository := storage.NewPostgresQualityRepository(pool)
	setpoints := defaultSetpoints()

	telemetryHandler := httphandler.NewTelemetryHandler(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpoints,
	)

	eventHandler := httphandler.NewEventHandler(logger, alertRepository)
	qualityHandler := httphandler.NewQualityHandler(logger, qualityRepository)

	mux := nethttp.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)

	mux.HandleFunc("/api/telemetry", telemetryHandler.Create)
	mux.HandleFunc("/api/telemetry/latest", telemetryHandler.Latest)
	mux.HandleFunc("/api/telemetry/history", telemetryHandler.History)

	mux.HandleFunc("/api/events", eventHandler.List)
	mux.HandleFunc("/api/events/active", eventHandler.Active)
	mux.HandleFunc("/api/events/", eventHandler.Action)

	mux.HandleFunc("/api/quality/latest", qualityHandler.Latest)
	mux.HandleFunc("/api/quality/history", qualityHandler.History)
	
	server := &nethttp.Server{
		Addr:              cfg.Server.Addr,
		Handler:           mux,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	logger.Info("starting server", "addr", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func defaultSetpoints() map[domain.ParameterType]domain.Setpoint {
	return map[domain.ParameterType]domain.Setpoint{
		domain.ParameterPressure: {
			ParameterType: domain.ParameterPressure,
			Unit:          domain.UnitBar,
			WarningMin:    30,
			NormalMin:     40,
			NormalMax:     75,
			WarningMax:    90,
		},
		domain.ParameterMoisture: {
			ParameterType: domain.ParameterMoisture,
			Unit:          domain.UnitPercent,
			WarningMin:    20,
			NormalMin:     22,
			NormalMax:     28,
			WarningMax:    30,
		},
		domain.ParameterBarrelTemperatureZone1: {
			ParameterType: domain.ParameterBarrelTemperatureZone1,
			Unit:          domain.UnitCelsius,
			WarningMin:    80,
			NormalMin:     90,
			NormalMax:     120,
			WarningMax:    130,
		},
		domain.ParameterBarrelTemperatureZone2: {
			ParameterType: domain.ParameterBarrelTemperatureZone2,
			Unit:          domain.UnitCelsius,
			WarningMin:    90,
			NormalMin:     100,
			NormalMax:     140,
			WarningMax:    150,
		},
		domain.ParameterBarrelTemperatureZone3: {
			ParameterType: domain.ParameterBarrelTemperatureZone3,
			Unit:          domain.UnitCelsius,
			WarningMin:    100,
			NormalMin:     110,
			NormalMax:     150,
			WarningMax:    160,
		},
		domain.ParameterScrewSpeed: {
			ParameterType: domain.ParameterScrewSpeed,
			Unit:          domain.UnitRPM,
			WarningMin:    150,
			NormalMin:     200,
			NormalMax:     450,
			WarningMax:    500,
		},
		domain.ParameterDriveLoad: {
			ParameterType: domain.ParameterDriveLoad,
			Unit:          domain.UnitPercent,
			WarningMin:    30,
			NormalMin:     40,
			NormalMax:     80,
			WarningMax:    90,
		},
		domain.ParameterOutletTemperature: {
			ParameterType: domain.ParameterOutletTemperature,
			Unit:          domain.UnitCelsius,
			WarningMin:    80,
			NormalMin:     90,
			NormalMax:     130,
			WarningMax:    140,
		},
	}
}
