package http

import (
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	nethttp "net/http"
)

// TelemetryCreateRequest describes the payload for telemetry ingestion.
type TelemetryCreateRequest = ingestion.TelemetryInput

// TelemetryCreateResponse describes the result of telemetry processing.
type TelemetryCreateResponse = ingestion.TelemetryResult

// TelemetryHandler handles telemetry API requests.
type TelemetryHandler struct {
	logger              *slog.Logger
	ingestionService    *ingestion.Service
	telemetryRepository storage.TelemetryRepository
	setpoints           map[domain.ParameterType]domain.Setpoint
}

// NewTelemetryHandler creates a telemetry HTTP handler.
func NewTelemetryHandler(
	logger *slog.Logger,
	telemetryRepository storage.TelemetryRepository,
	alertRepository storage.AlertRepository,
	qualityRepository storage.QualityRepository,
	setpoints map[domain.ParameterType]domain.Setpoint,
) *TelemetryHandler {
	ingestionService := ingestion.NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpoints,
	)

	return NewTelemetryHandlerWithService(
		logger,
		ingestionService,
		telemetryRepository,
		setpoints,
	)
}

// NewTelemetryHandlerWithService creates telemetry HTTP handler with existing ingestion service.
func NewTelemetryHandlerWithService(
	logger *slog.Logger,
	ingestionService *ingestion.Service,
	telemetryRepository storage.TelemetryRepository,
	setpoints map[domain.ParameterType]domain.Setpoint,
) *TelemetryHandler {
	return &TelemetryHandler{
		logger:              logger,
		ingestionService:    ingestionService,
		telemetryRepository: telemetryRepository,
		setpoints:           setpoints,
	}
}

// Create receives telemetry reading through HTTP and processes it.
func (h *TelemetryHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()

	var req TelemetryCreateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.ingestionService.Process(r.Context(), req)
	if err != nil {
		if ingestion.IsValidationError(err) {
			writeError(w, nethttp.StatusBadRequest, err.Error())
			return
		}

		h.logger.Error("process telemetry failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to process telemetry")
		return
	}

	writeJSON(w, nethttp.StatusCreated, result)
}
