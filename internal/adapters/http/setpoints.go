package httpadapter

import (
	"encoding/json"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ports"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"
)

type SetpointHandler struct {
	logger             *slog.Logger
	setpointRepository ports.SetpointRepository
}

func NewSetpointHandler(
	logger *slog.Logger,
	setpointRepository ports.SetpointRepository,
) *SetpointHandler {
	return &SetpointHandler{
		logger:             logger,
		setpointRepository: setpointRepository,
	}
}

func (h *SetpointHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	setpoints, err := h.setpointRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load setpoints failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load setpoints")
		return
	}

	writeJSON(w, nethttp.StatusOK, setpoints)
}

func (h *SetpointHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPut {
		w.Header().Set("Allow", nethttp.MethodPut)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := parseSetpointID(r.URL.Path)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	defer r.Body.Close()

	var req domain.SetpointUpdate

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeErrorWithDetails(
			w,
			nethttp.StatusBadRequest,
			"invalid_json_body",
			"invalid JSON body",
			map[string]string{
				"reason": err.Error(),
			},
		)
		return
	}

	if err := domain.ValidateSetpointUpdate(req); err != nil {
		writeErrorWithDetails(
			w,
			nethttp.StatusBadRequest,
			"validation_error",
			"invalid setpoint ranges",
			map[string]string{
				"reason": err.Error(),
			},
		)
		return
	}

	setpoint, found, err := h.setpointRepository.Update(r.Context(), id, req)
	if err != nil {
		h.logger.Error("update setpoint failed", "id", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to update setpoint")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "setpoint not found")
		return
	}

	updatedBy := "unknown"

	if user, ok := CurrentUser(r.Context()); ok {
		updatedBy = user.Username
	}

	h.logger.Info(
		"setpoint updated",
		"setpointId", setpoint.ID,
		"parameterType", setpoint.ParameterType,
		"updatedBy", updatedBy,
	)

	writeJSON(w, nethttp.StatusOK, setpoint)
}

func parseSetpointID(path string) (int64, error) {
	const prefix = "/api/setpoints/"

	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid setpoint path")
	}

	rawID := strings.TrimPrefix(path, prefix)

	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, fmt.Errorf("setpoint id is required")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid setpoint id")
	}

	return id, nil
}
