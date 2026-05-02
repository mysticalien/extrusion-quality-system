package httpadapter

import (
	"encoding/json"
	"net/http"
	"strings"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorResponse struct {
	Error APIError `json:"error"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeErrorWithCode(w, statusCode, errorCodeFromMessage(statusCode, message), message, nil)
}

func writeErrorWithDetails(w http.ResponseWriter, statusCode int, code string, message string, details any) {
	writeErrorWithCode(w, statusCode, code, message, details)
}

func writeErrorWithCode(
	w http.ResponseWriter,
	statusCode int,
	code string,
	message string,
	details any,
) {
	writeJSON(w, statusCode, errorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func errorCodeFromMessage(statusCode int, message string) string {
	normalized := strings.TrimSpace(strings.ToLower(message))

	switch normalized {
	case "invalid json body":
		return "invalid_json_body"

	case "method not allowed":
		return "method_not_allowed"

	case "missing authorization token":
		return "missing_authorization_token"

	case "invalid authorization token":
		return "invalid_authorization_token"

	case "authorization token expired":
		return "authorization_token_expired"

	case "unauthorized":
		return "unauthorized"

	case "forbidden":
		return "forbidden"

	case "not found":
		return "not_found"

	case "unknown parametertype":
		return "unknown_parameter_type"

	case "unit is required":
		return "unit_required"

	case "unit does not match parametertype":
		return "unit_mismatch"

	case "sourceid is required":
		return "source_id_required"

	case "measuredat is required":
		return "measured_at_required"

	case "setpoint id is required":
		return "setpoint_id_required"

	case "setpoint not found":
		return "setpoint_not_found"

	case "quality weight id is required":
		return "quality_weight_id_required"

	case "quality weight not found":
		return "quality_weight_not_found"

	case "weight must be positive":
		return "weight_must_be_positive"

	case "weight must not be greater than 10":
		return "weight_too_large"

	case "invalid username or password":
		return "invalid_username_or_password"

	case "user already exists":
		return "user_already_exists"

	case "invalid user role":
		return "invalid_user_role"

	case "old password is required":
		return "old_password_required"

	case "old password is incorrect":
		return "old_password_incorrect"

	case "new password must contain at least 12 characters":
		return "new_password_too_short"

	case "new password must be different from old password":
		return "new_password_same_as_old"
	}

	switch statusCode {
	case http.StatusBadRequest:
		return "bad_request"

	case http.StatusUnauthorized:
		return "unauthorized"

	case http.StatusForbidden:
		return "forbidden"

	case http.StatusNotFound:
		return "not_found"

	case http.StatusMethodNotAllowed:
		return "method_not_allowed"

	default:
		return "internal_error"
	}
}
