package http

import (
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	nethttp "net/http"
)

// QualityHandler handles quality index API requests.
type QualityHandler struct {
	logger       *slog.Logger
	qualityStore storage.QualityStore
}

// NewQualityHandler creates a quality index HTTP handler.
func NewQualityHandler(logger *slog.Logger, qualityStore storage.QualityStore) *QualityHandler {
	return &QualityHandler{
		logger:       logger,
		qualityStore: qualityStore,
	}
}

// Latest returns the latest calculated quality index.
func (h *QualityHandler) Latest(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/quality/latest" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	qualityIndex, found, err := h.qualityStore.Latest()
	if err != nil {
		h.logger.Error("load latest quality index failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load latest quality index")
		return
	}

	if !found {
		qualityIndex = analytics.CalculateQualityIndex(nil)
	}

	writeJSON(w, nethttp.StatusOK, qualityIndex)
}
