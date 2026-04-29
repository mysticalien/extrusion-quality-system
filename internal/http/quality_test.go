package http

import (
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestQualityHandlerLatest(t *testing.T) {
	tests := []struct {
		name                     string
		alerts                   []domain.AlertEvent
		expectedValue            float64
		expectedState            domain.QualityState
		expectedParameterPenalty float64
	}{
		{
			name:                     "without alerts returns stable quality index",
			alerts:                   nil,
			expectedValue:            100,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 0,
		},
		{
			name: "with warning alert returns decreased quality index",
			alerts: []domain.AlertEvent{
				{
					ParameterType: domain.ParameterPressure,
					Level:         domain.AlertLevelWarning,
					Status:        domain.AlertStatusActive,
					Value:         82.5,
					Unit:          domain.UnitBar,
					SourceID:      domain.SourceID("simulator"),
					Message:       "pressure warning",
				},
			},
			expectedValue:            85,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 15,
		},
		{
			name: "with critical alert returns stronger decreased quality index",
			alerts: []domain.AlertEvent{
				{
					ParameterType: domain.ParameterPressure,
					Level:         domain.AlertLevelCritical,
					Status:        domain.AlertStatusActive,
					Value:         95,
					Unit:          domain.UnitBar,
					SourceID:      domain.SourceID("simulator"),
					Message:       "pressure critical",
				},
			},
			expectedValue:            70,
			expectedState:            domain.QualityStateWarning,
			expectedParameterPenalty: 30,
		},
		{
			name: "resolved alert does not affect quality index",
			alerts: []domain.AlertEvent{
				{
					ParameterType: domain.ParameterPressure,
					Level:         domain.AlertLevelCritical,
					Status:        domain.AlertStatusResolved,
					Value:         95,
					Unit:          domain.UnitBar,
					SourceID:      domain.SourceID("simulator"),
					Message:       "pressure critical",
				},
			},
			expectedValue:            100,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			alertStore := storage.NewMemoryAlertStore()

			for _, alert := range tt.alerts {
				alertStore.Create(alert)
			}

			handler := NewQualityHandler(logger, alertStore)

			req := httptest.NewRequest(nethttp.MethodGet, "/api/quality/latest", nil)
			rec := httptest.NewRecorder()

			handler.Latest(rec, req)

			if rec.Code != nethttp.StatusOK {
				t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
			}

			var response domain.QualityIndex
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if response.Value != tt.expectedValue {
				t.Fatalf("expected value %.2f, got %.2f", tt.expectedValue, response.Value)
			}

			if response.State != tt.expectedState {
				t.Fatalf("expected state %q, got %q", tt.expectedState, response.State)
			}

			if response.ParameterPenalty != tt.expectedParameterPenalty {
				t.Fatalf(
					"expected parameterPenalty %.2f, got %.2f",
					tt.expectedParameterPenalty,
					response.ParameterPenalty,
				)
			}

			if response.AnomalyPenalty != 0 {
				t.Fatalf("expected anomalyPenalty 0, got %.2f", response.AnomalyPenalty)
			}

			if response.CalculatedAt.IsZero() {
				t.Fatalf("expected calculatedAt to be set")
			}
		})
	}
}

func TestQualityHandlerLatestInvalidRequests(t *testing.T) {
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
			path:           "/api/quality/latest",
			expectedStatus: nethttp.StatusMethodNotAllowed,
			expectedError:  "method not allowed",
		},
		{
			name:           "not found",
			method:         nethttp.MethodGet,
			path:           "/api/quality/unknown",
			expectedStatus: nethttp.StatusNotFound,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			alertStore := storage.NewMemoryAlertStore()
			handler := NewQualityHandler(logger, alertStore)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Latest(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.expectedError == "" {
				return
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
