package main

import (
	"extrusion-quality-system/internal/domain"
	httpapi "extrusion-quality-system/internal/http"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprint(w, "Homepage!")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	telemetryStore := storage.NewMemoryTelemetryStore()
	alertStore := storage.NewMemoryAlertStore()
	setpoints := defaultSetpoints()

	telemetryHandler := httpapi.NewTelemetryHandler(logger, telemetryStore, alertStore, setpoints)
	eventHandler := httpapi.NewEventHandler(logger, alertStore)
	qualityHandler := httpapi.NewQualityHandler(logger, alertStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/telemetry", telemetryHandler.Create)
	mux.HandleFunc("/api/events", eventHandler.List)
	mux.HandleFunc("/api/events/", eventHandler.Action)
	mux.HandleFunc("/api/quality/latest", qualityHandler.Latest)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
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
