package http

import (
	"encoding/json"
	"log/slog"
	nethttp "net/http"

	"extrusion-quality-system/internal/storage"
)

type SetpointHandler struct {
	logger             *slog.Logger
	setpointRepository storage.SetpointRepository
}

func NewSetpointHandler(
	logger *slog.Logger,
	setpointRepository storage.SetpointRepository,
) *SetpointHandler {
	return &SetpointHandler{
		logger:             logger,
		setpointRepository: setpointRepository,
	}
}

func (h *SetpointHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "method not allowed",
		})
		return
	}

	setpoints, err := h.setpointRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load setpoints failed", "error", err)

		w.WriteHeader(nethttp.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "failed to load setpoints",
		})
		return
	}

	if err := json.NewEncoder(w).Encode(setpoints); err != nil {
		h.logger.Error("write setpoints response failed", "error", err)
	}
}
