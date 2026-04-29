package http

import (
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// EventHandler handles alert event API requests.
type EventHandler struct {
	logger     *slog.Logger
	alertStore *storage.MemoryAlertStore
}

// NewEventHandler creates an alert event HTTP handler.
func NewEventHandler(logger *slog.Logger, alertStore *storage.MemoryAlertStore) *EventHandler {
	return &EventHandler{
		logger:     logger,
		alertStore: alertStore,
	}
}

// List returns all alert events.
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/events" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	events := h.alertStore.All()
	writeJSON(w, http.StatusOK, events)
}

// Action handles alert event actions, such as acknowledge and resolve.
func (h *EventHandler) Action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, action, ok := parseEventActionPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "ack":
		h.acknowledge(w, id)
	case "resolve":
		h.resolve(w, id)
	default:
		http.NotFound(w, r)
	}
}

func (h *EventHandler) acknowledge(w http.ResponseWriter, id domain.AlertID) {
	// UserID is not available yet because authentication is not implemented.
	// It will be passed here after role-based access control is added.
	event, ok := h.alertStore.Acknowledge(id, nil)
	if !ok {
		writeError(w, http.StatusNotFound, "alert not found")
		return
	}

	h.logger.Info("alert acknowledged", "alertId", id)
	writeJSON(w, http.StatusOK, event)
}

func (h *EventHandler) resolve(w http.ResponseWriter, id domain.AlertID) {
	event, ok := h.alertStore.Resolve(id)
	if !ok {
		writeError(w, http.StatusNotFound, "alert not found")
		return
	}

	h.logger.Info("alert resolved", "alertId", id)
	writeJSON(w, http.StatusOK, event)
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
