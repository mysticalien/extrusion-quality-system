package http

import (
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

func TestAnomalyHandlerActive(t *testing.T) {
	repository := storage.NewMemoryAnomalyRepository()
	handler := NewAnomalyHandler(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repository,
	)

	_, err := repository.Create(t.Context(), domain.AnomalyEvent{
		Type:          domain.AnomalyTypeCombinedRisk,
		ParameterType: domain.ParameterProcessRisk,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusActive,
		Message:       "combined risk",
		SourceID:      domain.SourceID("test-simulator"),
		ObservedAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create anomaly: %v", err)
	}

	_, err = repository.Create(t.Context(), domain.AnomalyEvent{
		Type:          domain.AnomalyTypeJump,
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusResolved,
		Message:       "resolved jump",
		SourceID:      domain.SourceID("test-simulator"),
		ObservedAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create resolved anomaly: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodGet, "/api/anomalies/active", nil)
	rec := httptest.NewRecorder()

	handler.Active(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response []domain.AnomalyEvent
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 active anomaly, got %d", len(response))
	}

	if response[0].Type != domain.AnomalyTypeCombinedRisk {
		t.Fatalf("expected anomaly type %q, got %q", domain.AnomalyTypeCombinedRisk, response[0].Type)
	}
}

func TestAnomalyHandlerActiveRejectsWrongMethod(t *testing.T) {
	repository := storage.NewMemoryAnomalyRepository()
	handler := NewAnomalyHandler(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repository,
	)

	req := httptest.NewRequest(nethttp.MethodPost, "/api/anomalies/active", nil)
	rec := httptest.NewRecorder()

	handler.Active(rec, req)

	if rec.Code != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", nethttp.StatusMethodNotAllowed, rec.Code)
	}
}
