package http

import (
	"bytes"
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	nethttp "net/http"
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
		expectedAlertLevel   domain.AlertLevel
		expectedQualityIndex float64
		expectedQualityState domain.QualityState
		expectedSavedCount   int
		expectedAlertCount   int
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
			expectedStatus:       nethttp.StatusCreated,
			expectedState:        domain.ParameterStateNormal,
			expectedAlertCreated: false,
			expectedAlertLevel:   "",
			expectedQualityIndex: 100,
			expectedQualityState: domain.QualityStateStable,
			expectedSavedCount:   1,
			expectedAlertCount:   0,
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
			expectedStatus:       nethttp.StatusCreated,
			expectedState:        domain.ParameterStateWarning,
			expectedAlertCreated: true,
			expectedAlertLevel:   domain.AlertLevelWarning,
			expectedQualityIndex: 85,
			expectedQualityState: domain.QualityStateStable,
			expectedSavedCount:   1,
			expectedAlertCount:   1,
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
			expectedStatus:       nethttp.StatusCreated,
			expectedState:        domain.ParameterStateCritical,
			expectedAlertCreated: true,
			expectedAlertLevel:   domain.AlertLevelCritical,
			expectedQualityIndex: 70,
			expectedQualityState: domain.QualityStateWarning,
			expectedSavedCount:   1,
			expectedAlertCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, telemetryStore, alertStore := newTestTelemetryHandler()

			req := httptest.NewRequest(
				nethttp.MethodPost,
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
				t.Fatalf("expected qualityIndex %.2f, got %.2f", tt.expectedQualityIndex, response.QualityIndex)
			}

			if response.QualityState != tt.expectedQualityState {
				t.Fatalf("expected qualityState %q, got %q", tt.expectedQualityState, response.QualityState)
			}

			if tt.expectedAlertCreated {
				if response.AlertID == nil {
					t.Fatalf("expected alertId to be set")
				}

				if response.AlertLevel == nil {
					t.Fatalf("expected alertLevel to be set")
				}

				if *response.AlertLevel != tt.expectedAlertLevel {
					t.Fatalf("expected alertLevel %q, got %q", tt.expectedAlertLevel, *response.AlertLevel)
				}
			} else {
				if response.AlertID != nil {
					t.Fatalf("expected alertId to be nil, got %d", *response.AlertID)
				}

				if response.AlertLevel != nil {
					t.Fatalf("expected alertLevel to be nil, got %q", *response.AlertLevel)
				}
			}

			savedReadings := telemetryStore.All()
			if len(savedReadings) != tt.expectedSavedCount {
				t.Fatalf("expected %d saved readings, got %d", tt.expectedSavedCount, len(savedReadings))
			}

			alerts := alertStore.All()
			if len(alerts) != tt.expectedAlertCount {
				t.Fatalf("expected %d stored alerts, got %d", tt.expectedAlertCount, len(alerts))
			}

			if tt.expectedAlertCreated {
				alert := alerts[0]

				if alert.Level != tt.expectedAlertLevel {
					t.Fatalf("expected stored alert level %q, got %q", tt.expectedAlertLevel, alert.Level)
				}

				if alert.Status != domain.AlertStatusActive {
					t.Fatalf("expected stored alert status %q, got %q", domain.AlertStatusActive, alert.Status)
				}

				if alert.ParameterType != domain.ParameterPressure {
					t.Fatalf("expected stored alert parameterType %q, got %q", domain.ParameterPressure, alert.ParameterType)
				}
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
			method:         nethttp.MethodGet,
			requestBody:    "",
			expectedStatus: nethttp.StatusMethodNotAllowed,
			expectedError:  "method not allowed",
		},
		{
			name:           "invalid JSON body",
			method:         nethttp.MethodPost,
			requestBody:    `{`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "invalid JSON body",
		},
		{
			name:   "unknown parameter type",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "unknown_parameter",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "unknown parameterType",
		},
		{
			name:   "unit is required",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "unit is required",
		},
		{
			name:   "unit does not match parameter type",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "percent",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "unit does not match parameterType",
		},
		{
			name:   "source id is required",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"measuredAt": "2026-04-27T18:00:00Z"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "sourceId is required",
		},
		{
			name:   "measured at is required",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "measuredAt is required",
		},
		{
			name:   "unknown field is rejected",
			method: nethttp.MethodPost,
			requestBody: `{
				"parameterType": "pressure",
				"value": 65,
				"unit": "bar",
				"sourceId": "simulator",
				"measuredAt": "2026-04-27T18:00:00Z",
				"extra": "not allowed"
			}`,
			expectedStatus: nethttp.StatusBadRequest,
			expectedError:  "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, telemetryStore, alertStore := newTestTelemetryHandler()

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

			if len(telemetryStore.All()) != 0 {
				t.Fatalf("expected no saved readings for invalid request")
			}

			if len(alertStore.All()) != 0 {
				t.Fatalf("expected no stored alerts for invalid request")
			}
		})
	}
}

func newTestTelemetryHandler() (*TelemetryHandler, *storage.MemoryTelemetryStore, *storage.MemoryAlertStore) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	telemetryStore := storage.NewMemoryTelemetryStore()
	alertStore := storage.NewMemoryAlertStore()

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

	return NewTelemetryHandler(logger, telemetryStore, alertStore, setpoints), telemetryStore, alertStore
}
