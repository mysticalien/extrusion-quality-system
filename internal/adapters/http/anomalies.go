package httpadapter

import (
	"encoding/json"
	"log/slog"
	nethttp "net/http"

	"extrusion-quality-system/internal/storage"
)

type AnomalyHandler struct {
	logger            *slog.Logger
	anomalyRepository storage.AnomalyRepository
}

func NewAnomalyHandler(
	logger *slog.Logger,
	anomalyRepository storage.AnomalyRepository,
) *AnomalyHandler {
	return &AnomalyHandler{
		logger:            logger,
		anomalyRepository: anomalyRepository,
	}
}

func (h *AnomalyHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	anomalies, err := h.anomalyRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load anomaly events failed", "error", err)

		w.WriteHeader(nethttp.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load anomaly events"})
		return
	}

	_ = json.NewEncoder(w).Encode(anomalies)
}

func (h *AnomalyHandler) Active(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	anomalies, err := h.anomalyRepository.Active(r.Context())
	if err != nil {
		h.logger.Error("load active anomaly events failed", "error", err)

		w.WriteHeader(nethttp.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load active anomaly events"})
		return
	}

	_ = json.NewEncoder(w).Encode(anomalies)
}
