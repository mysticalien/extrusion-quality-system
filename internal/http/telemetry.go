package http

import (
	"encoding/json"
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// TelemetryCreateRequest describes the payload for telemetry ingestion.
type TelemetryCreateRequest struct {
	ParameterType domain.ParameterType `json:"parameterType"`
	Value         float64              `json:"value"`
	Unit          domain.Unit          `json:"unit"`
	SourceID      domain.SourceID      `json:"sourceId"`
	MeasuredAt    time.Time            `json:"measuredAt"`
}

// TelemetryCreateResponse describes the result of telemetry processing.
type TelemetryCreateResponse struct {
	Accepted      bool                  `json:"accepted"`
	ParameterType domain.ParameterType  `json:"parameterType"`
	Value         float64               `json:"value"`
	Unit          domain.Unit           `json:"unit"`
	SourceID      domain.SourceID       `json:"sourceId"`
	MeasuredAt    time.Time             `json:"measuredAt"`
	State         domain.ParameterState `json:"state"`
	AlertCreated  bool                  `json:"alertCreated"`
	AlertID       *domain.AlertID       `json:"alertId,omitempty"`
	AlertLevel    *domain.AlertLevel    `json:"alertLevel,omitempty"`
	QualityIndex  float64               `json:"qualityIndex"`
	QualityState  domain.QualityState   `json:"qualityState"`
}

// TelemetryHandler handles telemetry API requests.
type TelemetryHandler struct {
	logger     *slog.Logger
	store      *storage.MemoryTelemetryStore
	alertStore *storage.MemoryAlertStore
	setpoints  map[domain.ParameterType]domain.Setpoint
}

// NewTelemetryHandler creates a telemetry HTTP handler.
func NewTelemetryHandler(
	logger *slog.Logger,
	store *storage.MemoryTelemetryStore,
	alertStore *storage.MemoryAlertStore,
	setpoints map[domain.ParameterType]domain.Setpoint,
) *TelemetryHandler {
	return &TelemetryHandler{
		logger:     logger,
		store:      store,
		alertStore: alertStore,
		setpoints:  setpoints,
	}
}

// Create receives a telemetry reading, stores it in memory, evaluates setpoints,
// creates an alert event if needed, and returns the processing result.
func (h *TelemetryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()

	var req TelemetryCreateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	setpoint, ok := h.setpoints[req.ParameterType]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown parameterType")
		return
	}

	if req.Unit == "" {
		writeError(w, http.StatusBadRequest, "unit is required")
		return
	}

	if req.Unit != setpoint.Unit {
		writeError(w, http.StatusBadRequest, "unit does not match parameterType")
		return
	}

	if req.SourceID == "" {
		writeError(w, http.StatusBadRequest, "sourceId is required")
		return
	}

	if req.MeasuredAt.IsZero() {
		writeError(w, http.StatusBadRequest, "measuredAt is required")
		return
	}

	reading := domain.TelemetryReading{
		ParameterType: req.ParameterType,
		Value:         req.Value,
		Unit:          req.Unit,
		SourceID:      req.SourceID,
		MeasuredAt:    req.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
	}

	reading = h.store.Save(reading)

	state := setpoint.Evaluate(reading.Value)

	var alertID *domain.AlertID
	var alertLevel *domain.AlertLevel

	level, shouldCreateAlert := alertLevelFromParameterState(state)
	if shouldCreateAlert {
		alert := domain.AlertEvent{
			ParameterType: reading.ParameterType,
			Level:         level,
			Status:        domain.AlertStatusActive,
			Value:         reading.Value,
			Unit:          reading.Unit,
			SourceID:      reading.SourceID,
			Message:       buildAlertMessage(reading, level),
			CreatedAt:     time.Now().UTC(),
		}

		alert = h.alertStore.Create(alert)

		id := alert.ID
		alertID = &id

		levelCopy := alert.Level
		alertLevel = &levelCopy
	}

	qualityIndex := analytics.CalculateQualityIndex(h.alertStore.Active())

	h.logger.Info(
		"telemetry received",
		"parameterType", reading.ParameterType,
		"value", reading.Value,
		"unit", reading.Unit,
		"sourceId", reading.SourceID,
		"state", state,
		"alertCreated", shouldCreateAlert,
		"qualityIndex", qualityIndex.Value,
		"qualityState", qualityIndex.State,
	)

	response := TelemetryCreateResponse{
		Accepted:      true,
		ParameterType: reading.ParameterType,
		Value:         reading.Value,
		Unit:          reading.Unit,
		SourceID:      reading.SourceID,
		MeasuredAt:    reading.MeasuredAt,
		State:         state,
		AlertCreated:  shouldCreateAlert,
		AlertID:       alertID,
		AlertLevel:    alertLevel,
		QualityIndex:  qualityIndex.Value,
		QualityState:  qualityIndex.State,
	}

	writeJSON(w, http.StatusCreated, response)
}

func alertLevelFromParameterState(state domain.ParameterState) (domain.AlertLevel, bool) {
	switch state {
	case domain.ParameterStateWarning:
		return domain.AlertLevelWarning, true
	case domain.ParameterStateCritical:
		return domain.AlertLevelCritical, true
	default:
		return "", false
	}
}

func buildAlertMessage(reading domain.TelemetryReading, level domain.AlertLevel) string {
	return fmt.Sprintf(
		"parameter %s has %s value %.2f %s",
		reading.ParameterType,
		level,
		reading.Value,
		reading.Unit,
	)
}
