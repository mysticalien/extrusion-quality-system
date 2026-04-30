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

func TestQualityHandlerLatest(t *testing.T) {
	tests := []struct {
		name                     string
		qualityIndex             *domain.QualityIndex
		expectedValue            float64
		expectedState            domain.QualityState
		expectedParameterPenalty float64
	}{
		{
			name:                     "without stored quality index returns default stable index",
			qualityIndex:             nil,
			expectedValue:            100,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 0,
		},
		{
			name: "with stored warning quality index returns latest value",
			qualityIndex: &domain.QualityIndex{
				Value:            85,
				State:            domain.QualityStateStable,
				ParameterPenalty: 15,
				AnomalyPenalty:   0,
				CalculatedAt:     time.Now().UTC(),
			},
			expectedValue:            85,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 15,
		},
		{
			name: "with stored critical quality index returns latest value",
			qualityIndex: &domain.QualityIndex{
				Value:            55,
				State:            domain.QualityStateUnstable,
				ParameterPenalty: 45,
				AnomalyPenalty:   0,
				CalculatedAt:     time.Now().UTC(),
			},
			expectedValue:            55,
			expectedState:            domain.QualityStateUnstable,
			expectedParameterPenalty: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			qualityRepository := storage.NewMemoryQualityRepository()

			if tt.qualityIndex != nil {
				_, err := qualityRepository.Save(ctx, *tt.qualityIndex)
				if err != nil {
					t.Fatalf("save quality index: %v", err)
				}
			}

			handler := NewQualityHandler(logger, qualityRepository)

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
			qualityRepository := storage.NewMemoryQualityRepository()
			handler := NewQualityHandler(logger, qualityRepository)

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
