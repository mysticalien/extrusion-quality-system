package httpadapter

import (
	"extrusion-quality-system/internal/ports"
	"log/slog"
	nethttp "net/http"
)

type AnomalyHandler struct {
	logger            *slog.Logger
	anomalyRepository ports.AnomalyRepository
}

func NewAnomalyHandler(
	logger *slog.Logger,
	anomalyRepository ports.AnomalyRepository,
) *AnomalyHandler {
	return &AnomalyHandler{
		logger:            logger,
		anomalyRepository: anomalyRepository,
	}
}

func (h *AnomalyHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	anomalies, err := h.anomalyRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load anomaly events failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load anomaly events")
		return
	}

	writeJSON(w, nethttp.StatusOK, anomalies)
}

func (h *AnomalyHandler) Active(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	anomalies, err := h.anomalyRepository.Active(r.Context())
	if err != nil {
		h.logger.Error("load active anomaly events failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load active anomaly events")
		return
	}

	writeJSON(w, nethttp.StatusOK, anomalies)
}
