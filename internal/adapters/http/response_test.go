package httpadapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorUsesUnifiedFormat(t *testing.T) {
	response := httptest.NewRecorder()

	writeError(response, http.StatusForbidden, "forbidden")

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}

	var body errorResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Error.Code != "forbidden" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "forbidden")
	}

	if body.Error.Message != "forbidden" {
		t.Fatalf("message = %q, want %q", body.Error.Message, "forbidden")
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	response := httptest.NewRecorder()

	writeErrorWithDetails(
		response,
		http.StatusBadRequest,
		"validation_error",
		"invalid telemetry input",
		map[string]string{
			"field":  "sourceId",
			"reason": "sourceId is required",
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var body errorResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Error.Code != "validation_error" {
		t.Fatalf("error code = %q, want validation_error", body.Error.Code)
	}

	if body.Error.Message != "invalid telemetry input" {
		t.Fatalf("message = %q, want invalid telemetry input", body.Error.Message)
	}

	details, ok := body.Error.Details.(map[string]any)
	if !ok {
		t.Fatalf("details type = %T, want map[string]any", body.Error.Details)
	}

	if details["field"] != "sourceId" {
		t.Fatalf("details field = %v, want sourceId", details["field"])
	}
}

func TestErrorCodeFromMessage(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		want       string
	}{
		{
			name:       "invalid json body",
			statusCode: http.StatusBadRequest,
			message:    "invalid JSON body",
			want:       "invalid_json_body",
		},
		{
			name:       "missing authorization token",
			statusCode: http.StatusUnauthorized,
			message:    "missing authorization token",
			want:       "missing_authorization_token",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			message:    "forbidden",
			want:       "forbidden",
		},
		{
			name:       "quality weight not found",
			statusCode: http.StatusNotFound,
			message:    "quality weight not found",
			want:       "quality_weight_not_found",
		},
		{
			name:       "fallback bad request",
			statusCode: http.StatusBadRequest,
			message:    "some unknown bad request",
			want:       "bad_request",
		},
		{
			name:       "fallback internal error",
			statusCode: http.StatusInternalServerError,
			message:    "unexpected error",
			want:       "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorCodeFromMessage(tt.statusCode, tt.message)

			if got != tt.want {
				t.Fatalf("code = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteJSONSetsContentTypeAndStatus(t *testing.T) {
	response := httptest.NewRecorder()

	writeJSON(response, http.StatusCreated, map[string]string{
		"status": "ok",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	contentType := response.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json; charset=utf-8", contentType)
	}
}
