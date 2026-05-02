package httpadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

type QualityWeightHandler struct {
	logger                  *slog.Logger
	qualityWeightRepository storage.QualityWeightRepository
}

func NewQualityWeightHandler(
	logger *slog.Logger,
	qualityWeightRepository storage.QualityWeightRepository,
) *QualityWeightHandler {
	return &QualityWeightHandler{
		logger:                  logger,
		qualityWeightRepository: qualityWeightRepository,
	}
}

func (h *QualityWeightHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.qualityWeightRepository == nil {
		writeError(w, nethttp.StatusInternalServerError, "quality weight repository is not configured")
		return
	}

	weights, err := h.qualityWeightRepository.List(r.Context())
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, "failed to load quality weights")
		return
	}

	writeJSON(w, nethttp.StatusOK, weights)
}

func (h *QualityWeightHandler) Update(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPut {
		w.Header().Set("Allow", nethttp.MethodPut)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.qualityWeightRepository == nil {
		writeError(w, nethttp.StatusInternalServerError, "quality weight repository is not configured")
		return
	}

	id, err := parseQualityWeightID(r.URL.Path)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	defer r.Body.Close()

	var req domain.QualityWeightUpdate

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

	if err := domain.ValidateQualityWeightUpdate(req); err != nil {
		writeErrorWithDetails(
			w,
			nethttp.StatusBadRequest,
			"validation_error",
			"invalid quality weight",
			map[string]string{
				"reason": err.Error(),
			},
		)
		return
	}

	updatedBy := "unknown"

	user, ok := CurrentUser(r.Context())
	if ok {
		updatedBy = user.Username
	}

	weight, found, err := h.qualityWeightRepository.Update(
		r.Context(),
		id,
		req,
		updatedBy,
	)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, "failed to update quality weight")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "quality weight not found")
		return
	}

	h.logger.Info(
		"quality weight updated",
		"qualityWeightId", weight.ID,
		"parameterType", weight.ParameterType,
		"weight", weight.Weight,
		"updatedBy", updatedBy,
	)

	writeJSON(w, nethttp.StatusOK, weight)
}

func parseQualityWeightID(path string) (domain.QualityWeightID, error) {
	const prefix = "/api/quality/weights/"

	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("quality weight id is required")
	}

	rawID := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rawID == "" {
		return 0, errors.New("quality weight id is required")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid quality weight id %q", rawID)
	}

	return domain.QualityWeightID(id), nil
}
