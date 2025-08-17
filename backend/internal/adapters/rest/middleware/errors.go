package middleware

import (
	"encoding/json"
	"net/http"
)

// Error codes used by middleware (lower_snake_case convention)
const (
	ErrorCodeUnauthorized       = "unauthorized"
	ErrorCodeForbidden          = "forbidden"
	ErrorCodeNotFound           = "not_found"
	ErrorCodeValidationError    = "validation_error"
	ErrorCodeInvalidToken       = "invalid_token"
	ErrorCodeTokenExpired       = "token_expired"
	ErrorCodeInternalServerError = "internal_server_error"
)

// WriteJSONError writes a JSON error response with consistent format
// This matches the format used by BaseHandler in the REST layer
func WriteJSONError(w http.ResponseWriter, code string, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	errorResp := map[string]any{
		"error":   code,
		"message": message,
	}
	
	// Ignore encoding errors here as we're already in error handling
	_ = json.NewEncoder(w).Encode(errorResp)
}

// WriteJSONErrorWithDetails writes a JSON error response with additional details
func WriteJSONErrorWithDetails(w http.ResponseWriter, code string, message string, status int, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	errorResp := map[string]any{
		"error":   code,
		"message": message,
	}
	
	// Add any additional details
	for k, v := range details {
		errorResp[k] = v
	}
	
	// Ignore encoding errors here as we're already in error handling
	_ = json.NewEncoder(w).Encode(errorResp)
}