package http

import (
	"encoding/json"
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

// Active returns active and acknowledged alert events.
func (h *EventHandler) Active(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/events/active" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	events, err := h.alertRepository.Active(r.Context())
	if err != nil {
		h.logger.Error("load active alert events failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load active alert events")
		return
	}

	writeJSON(w, nethttp.StatusOK, events)
}

// Action handles alert event actions, such as acknowledge and resolve.
func (h *EventHandler) Action(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	alertID, action, ok := parseEventActionPath(r.URL.Path)
	if !ok {
		w.WriteHeader(nethttp.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}

	switch action {
	case "ack":
		h.acknowledge(w, r, alertID)

	case "resolve":
		if !canResolveEvent(w, r) {
			return
		}

		h.resolve(w, r, alertID)

	default:
		w.WriteHeader(nethttp.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}
}

func canResolveEvent(w nethttp.ResponseWriter, r *nethttp.Request) bool {
	user, ok := CurrentUser(r.Context())
	if !ok {
		w.WriteHeader(nethttp.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return false
	}

	if user.Role != domain.UserRoleTechnologist && user.Role != domain.UserRoleAdmin {
		w.WriteHeader(nethttp.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return false
	}

	return true
}
