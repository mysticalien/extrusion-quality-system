package httpadapter

import (
	"net/http"
	"strings"
)

var errorCodesByMessage = map[string]string{
	"invalid json body":                                "invalid_json_body",
	"method not allowed":                               "method_not_allowed",
	"missing authorization token":                      "missing_authorization_token",
	"invalid authorization token":                      "invalid_authorization_token",
	"authorization token expired":                      "authorization_token_expired",
	"unauthorized":                                     "unauthorized",
	"forbidden":                                        "forbidden",
	"not found":                                        "not_found",
	"unknown parametertype":                            "unknown_parameter_type",
	"unit is required":                                 "unit_required",
	"unit does not match parametertype":                "unit_mismatch",
	"sourceid is required":                             "source_id_required",
	"measuredat is required":                           "measured_at_required",
	"setpoint id is required":                          "setpoint_id_required",
	"setpoint not found":                               "setpoint_not_found",
	"quality weight id is required":                    "quality_weight_id_required",
	"quality weight not found":                         "quality_weight_not_found",
	"weight must be positive":                          "weight_must_be_positive",
	"weight must not be greater than 10":               "weight_too_large",
	"invalid username or password":                     "invalid_username_or_password",
	"user already exists":                              "user_already_exists",
	"invalid user role":                                "invalid_user_role",
	"old password is required":                         "old_password_required",
	"old password is incorrect":                        "old_password_incorrect",
	"new password must contain at least 12 characters": "new_password_too_short",
	"new password must be different from old password": "new_password_same_as_old",
	"user is inactive or not found":                    "user_inactive_or_not_found",
	"failed to login":                                  "login_failed",
	"failed to generate token":                         "token_generation_failed",
}

func errorCodeFromMessage(statusCode int, message string) string {
	normalized := strings.TrimSpace(strings.ToLower(message))

	if code, ok := errorCodesByMessage[normalized]; ok {
		return code
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
