package httpadapter

import (
	"extrusion-quality-system/internal/domain"
	nethttp "net/http"
)

// Latest returns the latest telemetry readings for all parameters.
func (h *TelemetryHandler) Latest(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/telemetry/latest" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	readings, err := h.telemetryRepository.Latest(r.Context())
	if err != nil {
		h.logger.Error("load latest telemetry readings failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load latest telemetry readings")
		return
	}

	writeJSON(w, nethttp.StatusOK, readings)
}

// History returns telemetry history for a selected parameter.
func (h *TelemetryHandler) History(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.URL.Path != "/api/telemetry/history" {
		nethttp.NotFound(w, r)
		return
	}

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()

	rawParameter := query.Get("parameter")
	if rawParameter == "" {
		rawParameter = query.Get("parameterType")
	}

	if rawParameter == "" {
		writeError(w, nethttp.StatusBadRequest, "parameter is required")
		return
	}

	parameterType := domain.ParameterType(rawParameter)
	_, found, err := h.setpointRepository.GetByParameter(r.Context(), parameterType)
	if err != nil {
		h.logger.Error(
			"load setpoint failed",
			"parameterType", parameterType,
			"error", err,
		)
		writeError(w, nethttp.StatusInternalServerError, "failed to load setpoint")
		return
	}

	if !found {
		writeError(w, nethttp.StatusBadRequest, "unknown parameter")
		return
	}

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

	readings, err := h.telemetryRepository.HistoryByParameter(
		r.Context(),
		parameterType,
		from,
		to,
		limit,
	)
	if err != nil {
		h.logger.Error(
			"load telemetry history failed",
			"parameterType", parameterType,
			"error", err,
		)
		writeError(w, nethttp.StatusInternalServerError, "failed to load telemetry history")
		return
	}

	writeJSON(w, nethttp.StatusOK, readings)
}
