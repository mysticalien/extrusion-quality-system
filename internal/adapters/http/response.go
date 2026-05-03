package httpadapter

import (
	"encoding/json"
	"net/http"
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
