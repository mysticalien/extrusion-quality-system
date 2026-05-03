package httpadapter

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/usecase/telemetry"
)

func TestTelemetryHandlerCreateSuccess(t *testing.T) {
	telemetryRepository := &httpFakeTelemetryRepository{}
	alertRepository := &httpFakeAlertRepository{}
	qualityRepository := &httpFakeQualityRepository{}
	setpointRepository := newHTTPFakeSetpointRepository()
	anomalyRepository := &httpFakeAnomalyRepository{}

	service := telemetry.NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	handler := NewTelemetryHandlerWithService(
		slog.Default(),
		service,
		telemetryRepository,
		setpointRepository,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/telemetry",
		strings.NewReader(`{
			"parameterType": "pressure",
			"value": 80,
			"unit": "bar",
			"sourceId": "test-source",
			"measuredAt": "2026-05-03T10:00:00Z"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var body telemetry.Result

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !body.Accepted {
		t.Fatal("expected accepted response")
	}

	if body.ParameterType != domain.ParameterPressure {
		t.Fatalf("parameterType = %q, want %q", body.ParameterType, domain.ParameterPressure)
	}
}

func TestTelemetryHandlerCreateInvalidJSON(t *testing.T) {
	handler := newTestTelemetryHandler()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/telemetry",
		strings.NewReader(`{"parameterType":`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var body errorResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Error.Code != "invalid_json_body" {
		t.Fatalf("error code = %q, want invalid_json_body", body.Error.Code)
	}
}

func TestTelemetryHandlerCreateValidationError(t *testing.T) {
	handler := newTestTelemetryHandler()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/telemetry",
		strings.NewReader(`{
			"parameterType": "unknown_parameter",
			"value": 80,
			"unit": "bar",
			"sourceId": "test-source",
			"measuredAt": "2026-05-03T10:00:00Z"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var body errorResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Error.Code != "validation_error" {
		t.Fatalf("error code = %q, want validation_error", body.Error.Code)
	}
}

func TestTelemetryHandlerCreateMethodNotAllowed(t *testing.T) {
	handler := newTestTelemetryHandler()

	request := httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func newTestTelemetryHandler() *TelemetryHandler {
	telemetryRepository := &httpFakeTelemetryRepository{}
	alertRepository := &httpFakeAlertRepository{}
	qualityRepository := &httpFakeQualityRepository{}
	setpointRepository := newHTTPFakeSetpointRepository()
	anomalyRepository := &httpFakeAnomalyRepository{}

	service := telemetry.NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	return NewTelemetryHandlerWithService(
		slog.Default(),
		service,
		telemetryRepository,
		setpointRepository,
	)
}

func newHTTPFakeSetpointRepository() *httpFakeSetpointRepository {
	return &httpFakeSetpointRepository{
		setpoints: map[domain.ParameterType]domain.Setpoint{
			domain.ParameterPressure: {
				ID:            1,
				ParameterType: domain.ParameterPressure,
				Unit:          domain.UnitBar,
				CriticalMin:   30,
				WarningMin:    35,
				NormalMin:     40,
				NormalMax:     75,
				WarningMax:    90,
				CriticalMax:   95,
			},
		},
	}
}

type httpFakeTelemetryRepository struct {
	nextID   domain.TelemetryReadingID
	readings []domain.TelemetryReading
}

func (r *httpFakeTelemetryRepository) Save(
	ctx context.Context,
	reading domain.TelemetryReading,
) (domain.TelemetryReading, error) {
	_ = ctx

	if r.nextID == 0 {
		r.nextID = 1
	}

	reading.ID = r.nextID
	r.nextID++

	r.readings = append(r.readings, reading)

	return reading, nil
}

func (r *httpFakeTelemetryRepository) All(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx
	return append([]domain.TelemetryReading(nil), r.readings...), nil
}

func (r *httpFakeTelemetryRepository) Latest(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx
	return append([]domain.TelemetryReading(nil), r.readings...), nil
}

func (r *httpFakeTelemetryRepository) HistoryByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.TelemetryReading, error) {
	_ = ctx
	_ = from
	_ = to

	result := make([]domain.TelemetryReading, 0)

	for _, reading := range r.readings {
		if reading.ParameterType == parameterType {
			result = append(result, reading)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result, nil
}

type httpFakeAlertRepository struct {
	nextID domain.AlertID
	alerts []domain.AlertEvent
}

func (r *httpFakeAlertRepository) Create(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, error) {
	_ = ctx

	if r.nextID == 0 {
		r.nextID = 1
	}

	alert.ID = r.nextID
	r.nextID++

	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}

	r.alerts = append(r.alerts, alert)

	return alert, nil
}

func (r *httpFakeAlertRepository) All(ctx context.Context) ([]domain.AlertEvent, error) {
	_ = ctx
	return append([]domain.AlertEvent(nil), r.alerts...), nil
}

func (r *httpFakeAlertRepository) Active(ctx context.Context) ([]domain.AlertEvent, error) {
	_ = ctx

	result := make([]domain.AlertEvent, 0)

	for _, alert := range r.alerts {
		if alert.Status == domain.AlertStatusActive || alert.Status == domain.AlertStatusAcknowledged {
			result = append(result, alert)
		}
	}

	return result, nil
}

func (r *httpFakeAlertRepository) FindOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	for index := len(r.alerts) - 1; index >= 0; index-- {
		alert := r.alerts[index]
		if alert.ParameterType == parameterType &&
			(alert.Status == domain.AlertStatusActive || alert.Status == domain.AlertStatusAcknowledged) {
			return alert, true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

func (r *httpFakeAlertRepository) UpdateOpen(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	for index := range r.alerts {
		if r.alerts[index].ID == alert.ID {
			r.alerts[index] = alert
			return alert, true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

func (r *httpFakeAlertRepository) ResolveOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (int64, error) {
	_ = ctx

	var count int64
	now := time.Now().UTC()

	for index := range r.alerts {
		if r.alerts[index].ParameterType == parameterType &&
			(r.alerts[index].Status == domain.AlertStatusActive || r.alerts[index].Status == domain.AlertStatusAcknowledged) {
			r.alerts[index].Status = domain.AlertStatusResolved
			r.alerts[index].ResolvedAt = &now
			count++
		}
	}

	return count, nil
}

func (r *httpFakeAlertRepository) Acknowledge(
	ctx context.Context,
	id domain.AlertID,
	userID *domain.UserID,
) (domain.AlertEvent, bool, error) {
	_ = ctx
	_ = userID

	for index := range r.alerts {
		if r.alerts[index].ID == id {
			r.alerts[index].Status = domain.AlertStatusAcknowledged
			return r.alerts[index], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

func (r *httpFakeAlertRepository) Resolve(
	ctx context.Context,
	id domain.AlertID,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	for index := range r.alerts {
		if r.alerts[index].ID == id {
			r.alerts[index].Status = domain.AlertStatusResolved
			return r.alerts[index], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

type httpFakeQualityRepository struct {
	nextID  domain.QualityIndexID
	indexes []domain.QualityIndex
}

func (r *httpFakeQualityRepository) Save(
	ctx context.Context,
	index domain.QualityIndex,
) (domain.QualityIndex, error) {
	_ = ctx

	if r.nextID == 0 {
		r.nextID = 1
	}

	index.ID = r.nextID
	r.nextID++

	r.indexes = append(r.indexes, index)

	return index, nil
}

func (r *httpFakeQualityRepository) Latest(ctx context.Context) (domain.QualityIndex, bool, error) {
	_ = ctx

	if len(r.indexes) == 0 {
		return domain.QualityIndex{}, false, nil
	}

	return r.indexes[len(r.indexes)-1], true, nil
}

func (r *httpFakeQualityRepository) History(
	ctx context.Context,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.QualityIndex, error) {
	_ = ctx
	_ = from
	_ = to
	_ = limit

	return append([]domain.QualityIndex(nil), r.indexes...), nil
}

type httpFakeSetpointRepository struct {
	setpoints map[domain.ParameterType]domain.Setpoint
}

func (r *httpFakeSetpointRepository) All(ctx context.Context) ([]domain.Setpoint, error) {
	_ = ctx

	result := make([]domain.Setpoint, 0, len(r.setpoints))
	for _, setpoint := range r.setpoints {
		result = append(result, setpoint)
	}

	return result, nil
}

func (r *httpFakeSetpointRepository) GetByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.Setpoint, bool, error) {
	_ = ctx

	setpoint, found := r.setpoints[parameterType]

	return setpoint, found, nil
}

func (r *httpFakeSetpointRepository) Update(
	ctx context.Context,
	id int64,
	update domain.SetpointUpdate,
) (domain.Setpoint, bool, error) {
	_ = ctx

	for parameterType, setpoint := range r.setpoints {
		if int64(setpoint.ID) != id {
			continue
		}

		setpoint.CriticalMin = update.CriticalMin
		setpoint.WarningMin = update.WarningMin
		setpoint.NormalMin = update.NormalMin
		setpoint.NormalMax = update.NormalMax
		setpoint.WarningMax = update.WarningMax
		setpoint.CriticalMax = update.CriticalMax

		r.setpoints[parameterType] = setpoint

		return setpoint, true, nil
	}

	return domain.Setpoint{}, false, nil
}

type httpFakeAnomalyRepository struct {
	nextID    domain.AnomalyID
	anomalies []domain.AnomalyEvent
}

func (r *httpFakeAnomalyRepository) Create(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	_ = ctx

	if r.nextID == 0 {
		r.nextID = 1
	}

	anomaly.ID = r.nextID
	r.nextID++

	if anomaly.Status == "" {
		anomaly.Status = domain.AlertStatusActive
	}

	r.anomalies = append(r.anomalies, anomaly)

	return anomaly, nil
}

func (r *httpFakeAnomalyRepository) All(ctx context.Context) ([]domain.AnomalyEvent, error) {
	_ = ctx
	return append([]domain.AnomalyEvent(nil), r.anomalies...), nil
}

func (r *httpFakeAnomalyRepository) Active(ctx context.Context) ([]domain.AnomalyEvent, error) {
	_ = ctx

	result := make([]domain.AnomalyEvent, 0)

	for _, anomaly := range r.anomalies {
		if anomaly.Status == domain.AlertStatusActive || anomaly.Status == domain.AlertStatusAcknowledged {
			result = append(result, anomaly)
		}
	}

	return result, nil
}

func (r *httpFakeAnomalyRepository) FindOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (domain.AnomalyEvent, bool, error) {
	_ = ctx

	for index := len(r.anomalies) - 1; index >= 0; index-- {
		anomaly := r.anomalies[index]
		if anomaly.Type == anomalyType &&
			anomaly.ParameterType == parameterType &&
			(anomaly.Status == domain.AlertStatusActive || anomaly.Status == domain.AlertStatusAcknowledged) {
			return anomaly, true, nil
		}
	}

	return domain.AnomalyEvent{}, false, nil
}

func (r *httpFakeAnomalyRepository) UpdateOpen(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, bool, error) {
	_ = ctx

	for index := range r.anomalies {
		if r.anomalies[index].ID == anomaly.ID {
			r.anomalies[index] = anomaly
			return anomaly, true, nil
		}
	}

	return domain.AnomalyEvent{}, false, nil
}

func (r *httpFakeAnomalyRepository) ResolveOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (int64, error) {
	_ = ctx

	var count int64
	now := time.Now().UTC()

	for index := range r.anomalies {
		if r.anomalies[index].Type == anomalyType &&
			r.anomalies[index].ParameterType == parameterType &&
			(r.anomalies[index].Status == domain.AlertStatusActive || r.anomalies[index].Status == domain.AlertStatusAcknowledged) {
			r.anomalies[index].Status = domain.AlertStatusResolved
			r.anomalies[index].ResolvedAt = &now
			count++
		}
	}

	return count, nil
}
