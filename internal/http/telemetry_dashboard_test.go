package http

import (
	"context"
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTelemetryHandlerLatest(t *testing.T) {
	ctx := context.Background()
	handler, telemetryRepository := newTelemetryDashboardTestHandler(t)

	firstMeasuredAt := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	secondMeasuredAt := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)

	_, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save first pressure reading: %v", err)
	}

	expectedPressureReading, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    secondMeasuredAt,
		CreatedAt:     secondMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save latest pressure reading: %v", err)
	}

	expectedMoistureReading, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterMoisture,
		Value:         25,
		Unit:          domain.UnitPercent,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save moisture reading: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodGet, "/api/telemetry/latest", nil)
	rec := httptest.NewRecorder()

	handler.Latest(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response []domain.TelemetryReading
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 latest readings, got %d", len(response))
	}

	foundPressure := false
	foundMoisture := false

	for _, reading := range response {
		switch reading.ParameterType {
		case domain.ParameterPressure:
			foundPressure = true
			if reading.ID != expectedPressureReading.ID {
				t.Fatalf("expected latest pressure id %d, got %d", expectedPressureReading.ID, reading.ID)
			}
		case domain.ParameterMoisture:
			foundMoisture = true
			if reading.ID != expectedMoistureReading.ID {
				t.Fatalf("expected latest moisture id %d, got %d", expectedMoistureReading.ID, reading.ID)
			}
		}
	}

	if !foundPressure {
		t.Fatalf("expected pressure reading in response")
	}

	if !foundMoisture {
		t.Fatalf("expected moisture reading in response")
	}
}

func TestTelemetryHandlerLatestMethodNotAllowed(t *testing.T) {
	handler, _ := newTelemetryDashboardTestHandler(t)

	req := httptest.NewRequest(nethttp.MethodPost, "/api/telemetry/latest", nil)
	rec := httptest.NewRecorder()

	handler.Latest(rec, req)

	if rec.Code != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", nethttp.StatusMethodNotAllowed, rec.Code)
	}

	if rec.Header().Get("Allow") != nethttp.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", nethttp.MethodGet, rec.Header().Get("Allow"))
	}
}

func TestTelemetryHandlerHistory(t *testing.T) {
	ctx := context.Background()
	handler, telemetryRepository := newTelemetryDashboardTestHandler(t)

	firstMeasuredAt := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	secondMeasuredAt := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)
	thirdMeasuredAt := time.Date(2026, 4, 27, 18, 10, 0, 0, time.UTC)

	_, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save first pressure reading: %v", err)
	}

	expectedReading, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    secondMeasuredAt,
		CreatedAt:     secondMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save second pressure reading: %v", err)
	}

	_, err = telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         95,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    thirdMeasuredAt,
		CreatedAt:     thirdMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save third pressure reading: %v", err)
	}

	_, err = telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterMoisture,
		Value:         25,
		Unit:          domain.UnitPercent,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    secondMeasuredAt,
		CreatedAt:     secondMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save moisture reading: %v", err)
	}

	req := httptest.NewRequest(
		nethttp.MethodGet,
		"/api/telemetry/history?parameter=pressure&from=2026-04-27T18:05:00Z&to=2026-04-27T18:05:00Z&limit=10",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.History(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response []domain.TelemetryReading
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(response))
	}

	if response[0].ID != expectedReading.ID {
		t.Fatalf("expected reading id %d, got %d", expectedReading.ID, response[0].ID)
	}

	if response[0].ParameterType != domain.ParameterPressure {
		t.Fatalf("expected parameter %q, got %q", domain.ParameterPressure, response[0].ParameterType)
	}
}

func TestTelemetryHandlerHistoryLimit(t *testing.T) {
	ctx := context.Background()
	handler, telemetryRepository := newTelemetryDashboardTestHandler(t)

	firstMeasuredAt := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	secondMeasuredAt := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)

	_, err := telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save first reading: %v", err)
	}

	_, err = telemetryRepository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         70,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    secondMeasuredAt,
		CreatedAt:     secondMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save second reading: %v", err)
	}

	req := httptest.NewRequest(
		nethttp.MethodGet,
		"/api/telemetry/history?parameter=pressure&limit=1",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.History(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response []domain.TelemetryReading
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(response))
	}
}

func TestTelemetryHandlerHistoryInvalidRequests(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "method not allowed",
			method:         nethttp.MethodPost,
			path:           "/api/telemetry/history?parameter=pressure",
			expectedStatus: nethttp.StatusMethodNotAllowed,
			expectedError:  "method not allowed",
		},
		{
			name:           "parameter is required",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "parameter is required",
		},
		{
			name:           "unknown parameter",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=unknown",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "unknown parameter",
		},
		{
			name:           "invalid from",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=pressure&from=bad",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "from must be RFC3339 datetime",
		},
		{
			name:           "invalid to",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=pressure&to=bad",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "to must be RFC3339 datetime",
		},
		{
			name:           "from after to",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=pressure&from=2026-04-27T18:10:00Z&to=2026-04-27T18:00:00Z",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "from must be before or equal to to",
		},
		{
			name:           "invalid limit",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=pressure&limit=bad",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "limit must be integer",
		},
		{
			name:           "negative limit",
			method:         nethttp.MethodGet,
			path:           "/api/telemetry/history?parameter=pressure&limit=-1",
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "limit must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := newTelemetryDashboardTestHandler(t)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.History(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			var response errorResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}

			if response.Error != tt.expectedError {
				t.Fatalf("expected error %q, got %q", tt.expectedError, response.Error)
			}
		})
	}
}

func newTelemetryDashboardTestHandler(
	t *testing.T,
) (*TelemetryHandler, *storage.MemoryTelemetryRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	telemetryRepository := storage.NewMemoryTelemetryRepository()
	alertRepository := storage.NewMemoryAlertRepository()
	qualityRepository := storage.NewMemoryQualityRepository()

	setpoints := []domain.Setpoint{
		{
			ParameterType: domain.ParameterPressure,
			Unit:          domain.UnitBar,
			CriticalMin:   30,
			WarningMin:    35,
			NormalMin:     40,
			NormalMax:     75,
			WarningMax:    90,
			CriticalMax:   95,
		},
		{
			ParameterType: domain.ParameterMoisture,
			Unit:          domain.UnitPercent,
			CriticalMin:   15,
			WarningMin:    20,
			NormalMin:     22,
			NormalMax:     28,
			WarningMax:    30,
			CriticalMax:   35,
		},
	}

	handler := NewTelemetryHandler(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpoints,
	)

	return handler, telemetryRepository
}
