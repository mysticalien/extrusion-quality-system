package httpadapter

import (
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/storage"
	"log/slog"
	nethttp "net/http"
)

// QualityHandler handles quality index API requests.
type QualityHandler struct {
	logger            *slog.Logger
	qualityRepository storage.QualityRepository
}

// NewQualityHandler creates a quality index HTTP handler.
func NewQualityHandler(logger *slog.Logger, qualityRepository storage.QualityRepository) *QualityHandler {
	return &QualityHandler{
		logger:            logger,
		qualityRepository: qualityRepository,
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

	qualityIndex, found, err := h.qualityRepository.Latest(r.Context())
	if err != nil {
		h.logger.Error("load latest quality index failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load latest quality index")
		return
	}

	if !found {
		qualityIndex = analytics.CalculateQualityIndex(
			nil,
			analytics.DefaultQualityWeights(),
		)
	}

	writeJSON(w, nethttp.StatusOK, qualityIndex)
}

// History returns quality index history.
func (h *QualityHandler) History(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/quality/history" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()

	from, err := parseOptionalTimeParam(query, "from")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	to, err := parseOptionalTimeParam(query, "to")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	if err := validateTimeRange(from, to); err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	limit, err := parseLimitParam(query)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	history, err := h.qualityRepository.History(r.Context(), from, to, limit)
	if err != nil {
		h.logger.Error("load quality index history failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load quality index history")
		return
	}

	writeJSON(w, nethttp.StatusOK, history)
}
