package http

import (
	"bytes"
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelemetryHandlerCreate(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          string
		expectedStatus       int
		expectedState        domain.ParameterState
		expectedAlertCreated bool
		expectedQualityIndex int
		expectedSavedCount   int
	}{
		{
			name: "normal pressure reading",
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus:       http.StatusCreated,
			expectedState:        domain.ParameterStateNormal,
			expectedAlertCreated: false,
			expectedQualityIndex: 100,
			expectedSavedCount:   1,
		},
		{
			name: "warning pressure reading",
			requestBody: `{
				"parameterType": "pressure",
				"value": 82.5,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus:       http.StatusCreated,
			expectedState:        domain.ParameterStateWarning,
			expectedAlertCreated: true,
			expectedQualityIndex: 80,
			expectedSavedCount:   1,
		},
		{
			name: "critical pressure reading",
			requestBody: `{
				"parameterType": "pressure",
				"value": 95,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus:       http.StatusCreated,
			expectedState:        domain.ParameterStateCritical,
			expectedAlertCreated: true,
			expectedQualityIndex: 50,
			expectedSavedCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store := newTestTelemetryHandler()

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/telemetry",
				bytes.NewBufferString(tt.requestBody),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			var response TelemetryCreateResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if !response.Accepted {
				t.Fatalf("expected accepted response")
			}

			if response.ParameterType != domain.ParameterPressure {
				t.Fatalf("expected parameterType %q, got %q", domain.ParameterPressure, response.ParameterType)
			}

			if response.State != tt.expectedState {
				t.Fatalf("expected state %q, got %q", tt.expectedState, response.State)
			}

			if response.AlertCreated != tt.expectedAlertCreated {
				t.Fatalf("expected alertCreated %v, got %v", tt.expectedAlertCreated, response.AlertCreated)
			}

			if response.QualityIndex != tt.expectedQualityIndex {
				t.Fatalf("expected qualityIndex %d, got %d", tt.expectedQualityIndex, response.QualityIndex)
			}

			savedReadings := store.All()
			if len(savedReadings) != tt.expectedSavedCount {
				t.Fatalf("expected %d saved readings, got %d", tt.expectedSavedCount, len(savedReadings))
			}
		})
	}
}

func TestTelemetryHandlerCreateInvalidRequests(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			requestBody:    "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "method not allowed",
		},
		{
			name:           "invalid JSON body",
			method:         http.MethodPost,
			requestBody:    `{`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid JSON body",
		},
		{
			name:   "unknown parameter type",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "unknown_parameter",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "unknown parameterType",
		},
		{
			name:   "unit is required",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "unit is required",
		},
		{
			name:   "unit does not match parameter type",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "percent",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "unit does not match parameterType",
		},
		{
			name:   "source id is required",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "sourceId is required",
		},
		{
			name:   "measured at is required",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "measuredAt is required",
		},
		{
			name:   "unknown field is rejected",
			method: http.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z",
				"extra": "not allowed"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store := newTestTelemetryHandler()

			req := httptest.NewRequest(
				tt.method,
				"/api/telemetry",
				bytes.NewBufferString(tt.requestBody),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			handler.Create(rec, req)

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

			if len(store.All()) != 0 {
				t.Fatalf("expected no saved readings for invalid request")
			}
		})
	}
}

func newTestTelemetryHandler() (*TelemetryHandler, *storage.MemoryTelemetryStore) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := storage.NewMemoryTelemetryStore()

	setpoints := map[domain.ParameterType]domain.Setpoint{
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
	}

	return NewTelemetryHandler(logger, store, setpoints), store
}
