package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		message        string
		status         int
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "writes unauthorized error",
			code:           ErrorCodeUnauthorized,
			message:        "Authentication required",
			status:         http.StatusUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]string{
				"error":   "unauthorized",
				"message": "Authentication required",
			},
		},
		{
			name:           "writes forbidden error",
			code:           ErrorCodeForbidden,
			message:        "Insufficient permissions",
			status:         http.StatusForbidden,
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]string{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			},
		},
		{
			name:           "writes invalid token error",
			code:           ErrorCodeInvalidToken,
			message:        "Token is invalid",
			status:         http.StatusUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]string{
				"error":   "invalid_token",
				"message": "Token is invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test response writer
			w := httptest.NewRecorder()

			// Call the function
			WriteJSONError(w, tt.code, tt.message, tt.status)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if actualValue, ok := response[key]; !ok {
					t.Errorf("expected key %q not found in response", key)
				} else if actualValue != expectedValue {
					t.Errorf("for key %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestWriteJSONErrorWithDetails(t *testing.T) {
	// Create a test response writer
	w := httptest.NewRecorder()

	// Call the function with details
	details := map[string]any{
		"field":    "email",
		"required": true,
	}
	WriteJSONErrorWithDetails(w, ErrorCodeValidationError, "Validation failed", http.StatusBadRequest, details)

	// Check status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check standard fields
	if response["error"] != "validation_error" {
		t.Errorf("expected error code validation_error, got %v", response["error"])
	}
	if response["message"] != "Validation failed" {
		t.Errorf("expected message 'Validation failed', got %v", response["message"])
	}

	// Check details were included
	if response["field"] != "email" {
		t.Errorf("expected field 'email', got %v", response["field"])
	}
	if response["required"] != true {
		t.Errorf("expected required true, got %v", response["required"])
	}
}