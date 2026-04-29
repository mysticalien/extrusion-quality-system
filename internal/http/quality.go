package http

import (
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	"net/http"
)

// QualityHandler handles quality index API requests.
type QualityHandler struct {
	logger     *slog.Logger
	alertStore *storage.MemoryAlertStore
}

// NewQualityHandler creates a quality index HTTP handler.
func NewQualityHandler(logger *slog.Logger, alertStore *storage.MemoryAlertStore) *QualityHandler {
	return &QualityHandler{
		logger:     logger,
		alertStore: alertStore,
	}
}

// Latest returns the latest calculated quality index.
func (h *QualityHandler) Latest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/quality/latest" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	qualityIndex := analytics.CalculateQualityIndex(h.alertStore.Active())

	h.logger.Info(
		"quality index calculated",
		"value", qualityIndex.Value,
		"state", qualityIndex.State,
		"parameterPenalty", qualityIndex.ParameterPenalty,
		"anomalyPenalty", qualityIndex.AnomalyPenalty,
	)

	writeJSON(w, http.StatusOK, qualityIndex)
}
