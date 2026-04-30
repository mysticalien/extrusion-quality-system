package http

import (
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"
)

// EventHandler handles alert event API requests.
type EventHandler struct {
	logger          *slog.Logger
	alertRepository storage.AlertRepository
}

// NewEventHandler creates an alert event HTTP handler.
func NewEventHandler(logger *slog.Logger, alertRepository storage.AlertRepository) *EventHandler {
	return &EventHandler{
		logger:          logger,
		alertRepository: alertRepository,
	}
}

// List returns all alert events.
func (h *EventHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/events" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	events, err := h.alertRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load alert events failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load alert events")
		return
	}

	writeJSON(w, nethttp.StatusOK, events)
}

// Action handles alert event actions, such as acknowledge and resolve.
func (h *EventHandler) Action(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, action, ok := parseEventActionPath(r.URL.Path)
	if !ok {
		nethttp.NotFound(w, r)
		return
	}

	switch action {
	case "ack":
		h.acknowledge(w, r, id)
	case "resolve":
		h.resolve(w, r, id)
	default:
		nethttp.NotFound(w, r)
	}
}

func (h *EventHandler) acknowledge(w nethttp.ResponseWriter, r *nethttp.Request, id domain.AlertID) {
	event, found, err := h.alertRepository.Acknowledge(r.Context(), id, nil)
	if err != nil {
		h.logger.Error("acknowledge alert failed", "alertId", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "alert not found")
		return
	}

	h.logger.Info("alert acknowledged", "alertId", id)
	writeJSON(w, nethttp.StatusOK, event)
}

func (h *EventHandler) resolve(w nethttp.ResponseWriter, r *nethttp.Request, id domain.AlertID) {
	event, found, err := h.alertRepository.Resolve(r.Context(), id)
	if err != nil {
		h.logger.Error("resolve alert failed", "alertId", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to resolve alert")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "alert not found")
		return
	}

	h.logger.Info("alert resolved", "alertId", id)
	writeJSON(w, nethttp.StatusOK, event)
}

func parseEventActionPath(path string) (domain.AlertID, string, bool) {
	const prefix = "/api/events/"

	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")

	if len(parts) != 2 {
		return 0, "", false
	}

	rawID := parts[0]
	action := parts[1]

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}

	return domain.AlertID(id), action, true
}
