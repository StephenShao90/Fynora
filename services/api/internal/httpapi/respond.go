package httpapi

import (
	"encoding/json"
	"net/http"
)

const (
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeRateLimited     = "RATE_LIMITED"
	CodeInternalError   = "INTERNAL_ERROR"
)

var supportedErrorCodes = map[string]struct{}{
	CodeValidationError: {},
	CodeUnauthorized:    {},
	CodeForbidden:       {},
	CodeNotFound:        {},
	CodeConflict:        {},
	CodeRateLimited:     {},
	CodeInternalError:   {},
}

func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":      normalizeErrorCode(code),
			"message":   message,
			"requestId": r.Header.Get("X-Request-ID"),
		},
	})
}

func normalizeErrorCode(code string) string {
	if _, ok := supportedErrorCodes[code]; ok {
		return code
	}
	switch code {
	case "IDEMPOTENCY_CONFLICT":
		return CodeConflict
	case "INTERNAL", "DATABASE_ERROR", "DATABASE_UNAVAILABLE":
		return CodeInternalError
	}
	return CodeInternalError
}
