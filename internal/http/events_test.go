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

func TestEventHandlerList(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	_, err := alertRepository.Create(ctx, domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure warning",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var events []domain.AlertEvent
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].ID != 1 {
		t.Fatalf("expected event id 1, got %d", events[0].ID)
	}

	if events[0].Status != domain.AlertStatusActive {
		t.Fatalf("expected status %q, got %q", domain.AlertStatusActive, events[0].Status)
	}
}

func TestEventHandlerListMethodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	req := httptest.NewRequest(nethttp.MethodPost, "/api/events", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", nethttp.StatusMethodNotAllowed, rec.Code)
	}

	if rec.Header().Get("Allow") != nethttp.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", nethttp.MethodGet, rec.Header().Get("Allow"))
	}
}

func TestEventHandlerAcknowledge(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	alert, err := alertRepository.Create(ctx, domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure warning",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodPost, "/api/events/1/ack", nil)
	rec := httptest.NewRecorder()

	handler.Action(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response domain.AlertEvent
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != alert.ID {
		t.Fatalf("expected alert id %d, got %d", alert.ID, response.ID)
	}

	if response.Status != domain.AlertStatusAcknowledged {
		t.Fatalf("expected status %q, got %q", domain.AlertStatusAcknowledged, response.Status)
	}

	if response.AcknowledgedAt == nil {
		t.Fatalf("expected acknowledgedAt to be set")
	}
}

func TestEventHandlerResolve(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	alert, err := alertRepository.Create(ctx, domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusActive,
		Value:         95,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure critical",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodPost, "/api/events/1/resolve", nil)
	rec := httptest.NewRecorder()

	handler.Action(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusOK, rec.Code, rec.Body.String())
	}

	var response domain.AlertEvent
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != alert.ID {
		t.Fatalf("expected alert id %d, got %d", alert.ID, response.ID)
	}

	if response.Status != domain.AlertStatusResolved {
		t.Fatalf("expected status %q, got %q", domain.AlertStatusResolved, response.Status)
	}

	if response.ResolvedAt == nil {
		t.Fatalf("expected resolvedAt to be set")
	}
}

func TestEventHandlerActionNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "unknown alert id",
			path: "/api/events/999/ack",
		},
		{
			name: "invalid alert id",
			path: "/api/events/bad/ack",
		},
		{
			name: "unknown action",
			path: "/api/events/1/unknown",
		},
		{
			name: "invalid path",
			path: "/api/events/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(nethttp.MethodPost, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Action(rec, req)

			if rec.Code != nethttp.StatusNotFound {
				t.Fatalf("expected status %d, got %d, body: %s", nethttp.StatusNotFound, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestEventHandlerActionMethodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	alertRepository := storage.NewMemoryAlertRepository()
	handler := NewEventHandler(logger, alertRepository)

	req := httptest.NewRequest(nethttp.MethodGet, "/api/events/1/ack", nil)
	rec := httptest.NewRecorder()

	handler.Action(rec, req)

	if rec.Code != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", nethttp.StatusMethodNotAllowed, rec.Code)
	}

	if rec.Header().Get("Allow") != nethttp.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", nethttp.MethodPost, rec.Header().Get("Allow"))
	}
}
