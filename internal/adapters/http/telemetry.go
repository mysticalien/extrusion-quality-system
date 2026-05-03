package httpadapter

import (
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ports"
	"extrusion-quality-system/internal/usecase/telemetry"
	"log/slog"
	nethttp "net/http"
	"time"
)

type telemetryCreateRequest struct {
	ParameterType domain.ParameterType `json:"parameterType"`
	Value         float64              `json:"value"`
	Unit          domain.Unit          `json:"unit"`
	SourceID      domain.SourceID      `json:"sourceId"`
	MeasuredAt    time.Time            `json:"measuredAt"`
}

// TelemetryCreateRequest describes the payload for telemetry usecase.
type TelemetryCreateRequest = telemetry.Input

// TelemetryCreateResponse describes the result of telemetry processing.
type TelemetryCreateResponse = telemetry.Result

// TelemetryHandler handles telemetry API requests.
type TelemetryHandler struct {
	logger              *slog.Logger
	telemetryService    *telemetry.Service
	telemetryRepository ports.TelemetryRepository
	setpointRepository  ports.SetpointRepository
}

// NewTelemetryHandlerWithService creates telemetry HTTP handler with existing usecase service.
func NewTelemetryHandlerWithService(
	logger *slog.Logger,
	service *telemetry.Service,
	telemetryRepository ports.TelemetryRepository,
	setpointRepository ports.SetpointRepository,
) *TelemetryHandler {
	return &TelemetryHandler{
		logger:              logger,
		telemetryService:    service,
		telemetryRepository: telemetryRepository,
		setpointRepository:  setpointRepository,
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
		writeErrorWithDetails(
			w,
			nethttp.StatusBadRequest,
			"invalid_json_body",
			"invalid JSON body",
			map[string]string{
				"reason": err.Error(),
			},
		)
		return
	}

	result, err := h.telemetryService.Process(r.Context(), req)
	if err != nil {
		if telemetry.IsValidationError(err) {
			writeErrorWithDetails(
				w,
				nethttp.StatusBadRequest,
				"validation_error",
				"invalid telemetry input",
				validationDetailsFromMessage(err.Error()),
			)
			return
		}

		h.logger.Error("process telemetry failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to process telemetry")
		return
	}

	writeJSON(w, nethttp.StatusCreated, result)
}

func validationDetailsFromMessage(message string) map[string]string {
	switch message {
	case "unknown parameterType":
		return map[string]string{
			"field":  "parameterType",
			"reason": message,
		}

	case "unit is required":
		return map[string]string{
			"field":  "unit",
			"reason": message,
		}

	case "unit does not match parameterType":
		return map[string]string{
			"field":  "unit",
			"reason": message,
		}

	case "sourceId is required":
		return map[string]string{
			"field":  "sourceId",
			"reason": message,
		}

	case "measuredAt is required":
		return map[string]string{
			"field":  "measuredAt",
			"reason": message,
		}

	default:
		return map[string]string{
			"reason": message,
		}
	}
}

func (r telemetryCreateRequest) toUsecaseInput() telemetry.Input {
	return telemetry.Input{
		ParameterType: r.ParameterType,
		Value:         r.Value,
		Unit:          r.Unit,
		SourceID:      r.SourceID,
		MeasuredAt:    r.MeasuredAt,
	}
}
