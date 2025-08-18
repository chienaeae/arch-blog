package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/adapters/rest"
	"backend/internal/adapters/rest/middleware"
	"backend/internal/platform/apperror"
	"github.com/google/uuid"
)

// mockLogger implements the logger.Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name               string
		code               string
		message            string
		statusCode         int
		expectedBody       map[string]interface{}
		expectedStatusCode int
	}{
		{
			name:       "writes not found error",
			code:       "not_found",
			message:    "Resource not found",
			statusCode: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error":   "not_found",
				"message": "Resource not found",
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:       "writes validation error",
			code:       "validation_error",
			message:    "Invalid input",
			statusCode: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "validation_error",
				"message": "Invalid input",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:       "writes internal server error",
			code:       "internal_server_error",
			message:    "Something went wrong",
			statusCode: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error":   "internal_server_error",
				"message": "Something went wrong",
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base handler with mock logger
			handler := rest.NewBaseHandler(&mockLogger{})

			// Create test request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Call the method
			handler.WriteJSONError(rec, req, tt.code, tt.message, tt.statusCode)

			// Check status code
			if rec.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, rec.Code)
			}

			// Check content type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response body: %v", err)
			}

			// Check response fields
			if response["error"] != tt.expectedBody["error"] {
				t.Errorf("expected error %v, got %v", tt.expectedBody["error"], response["error"])
			}
			if response["message"] != tt.expectedBody["message"] {
				t.Errorf("expected message %v, got %v", tt.expectedBody["message"], response["message"])
			}
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	tests := []struct {
		name               string
		data               interface{}
		statusCode         int
		expectedStatusCode int
	}{
		{
			name: "writes success response with struct",
			data: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "123",
				Name: "Test User",
			},
			statusCode:         http.StatusOK,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "writes created response with map",
			data:               map[string]string{"status": "created"},
			statusCode:         http.StatusCreated,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "writes no content response with nil",
			data:               nil,
			statusCode:         http.StatusNoContent,
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base handler with mock logger
			handler := rest.NewBaseHandler(&mockLogger{})

			// Create test request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Call the method
			handler.WriteJSONResponse(rec, req, tt.data, tt.statusCode)

			// Check status code
			if rec.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, rec.Code)
			}

			// Check content type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// For non-nil data, verify it can be unmarshaled
			if tt.data != nil && rec.Body.Len() > 0 {
				var response interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedStatusCode int
		expectedError      string
		expectedBizCode    string
		expectedContext    interface{}
	}{
		{
			name: "handles AppError with business code",
			err: apperror.New(
				apperror.CodeNotFound,
				apperror.BusinessCodeUserNotFound,
				"user not found",
				http.StatusNotFound,
			),
			expectedStatusCode: http.StatusNotFound,
			expectedError:      "NOT_FOUND",
			expectedBizCode:    "USER_NOT_FOUND",
		},
		{
			name: "handles AppError with details",
			err: apperror.New(
				apperror.CodeValidationFailed,
				apperror.BusinessCodeInvalidEmail,
				"invalid email format",
				http.StatusBadRequest,
			).WithDetails(map[string]string{"field": "email"}),
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "VALIDATION_FAILED",
			expectedBizCode:    "INVALID_EMAIL",
			expectedContext:    map[string]interface{}{"field": "email"},
		},
		{
			name:               "handles unknown error as internal server error",
			err:                errors.New("unexpected error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "INTERNAL_SERVER_ERROR",
		},
		{
			name: "handles wrapped AppError",
			err: apperror.Wrap(
				errors.New("database error"),
				apperror.CodeInternalError,
				apperror.BusinessCodeGeneral,
				"failed to fetch data",
				http.StatusInternalServerError,
			),
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "INTERNAL_SERVER_ERROR",
			expectedBizCode:    "GENERAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base handler with mock logger
			handler := rest.NewBaseHandler(&mockLogger{})

			// Create test request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Call the method
			handler.HandleError(rec, req, tt.err)

			// Check status code
			if rec.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, rec.Code)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response body: %v", err)
			}

			// Check error code
			if response["error"] != tt.expectedError {
				t.Errorf("expected error code %v, got %v", tt.expectedError, response["error"])
			}

			// Check business code if expected
			if tt.expectedBizCode != "" {
				if response["business_code"] != tt.expectedBizCode {
					t.Errorf("expected business code %v, got %v", tt.expectedBizCode, response["business_code"])
				}
			}

			// Check context if expected
			if tt.expectedContext != nil {
				context, ok := response["context"]
				if !ok {
					t.Errorf("expected context in response but not found")
				} else {
					// Compare as JSON to handle type differences
					expectedJSON, _ := json.Marshal(tt.expectedContext)
					actualJSON, _ := json.Marshal(context)
					if string(expectedJSON) != string(actualJSON) {
						t.Errorf("expected context %s, got %s", expectedJSON, actualJSON)
					}
				}
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		paramName   string
		expectValid bool
		expectUUID  uuid.UUID
	}{
		{
			name:        "parses valid UUID",
			value:       "550e8400-e29b-41d4-a716-446655440000",
			paramName:   "user_id",
			expectValid: true,
			expectUUID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:        "rejects invalid UUID",
			value:       "not-a-uuid",
			paramName:   "role_id",
			expectValid: false,
			expectUUID:  uuid.Nil,
		},
		{
			name:        "rejects empty string",
			value:       "",
			paramName:   "id",
			expectValid: false,
			expectUUID:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base handler with mock logger
			handler := rest.NewBaseHandler(&mockLogger{})

			// Create test request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Call the method
			result, valid := handler.ParseUUID(rec, req, tt.value, tt.paramName)

			// Check validity
			if valid != tt.expectValid {
				t.Errorf("expected valid=%v, got %v", tt.expectValid, valid)
			}

			// Check UUID value
			if result != tt.expectUUID {
				t.Errorf("expected UUID %v, got %v", tt.expectUUID, result)
			}

			// If invalid, check error response
			if !tt.expectValid {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected status code 400 for invalid UUID, got %d", rec.Code)
				}

				var response map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to parse error response: %v", err)
				}

				if response["error"] != "invalid_request" {
					t.Errorf("expected error code 'invalid_request', got %v", response["error"])
				}

				expectedMessage := "Invalid " + tt.paramName
				if response["message"] != expectedMessage {
					t.Errorf("expected message %q, got %v", expectedMessage, response["message"])
				}
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectID    uuid.UUID
		shouldPanic bool
	}{
		{
			name: "retrieves user ID from context",
			setupCtx: func() context.Context {
				userID := uuid.New()
				ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
				return ctx
			},
			expectID: uuid.New(), // Will be set in test
		},
		{
			name: "panics when user ID not in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base handler with mock logger
			handler := rest.NewBaseHandler(&mockLogger{})

			// Setup context
			ctx := tt.setupCtx()
			req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

			if tt.shouldPanic {
				// Test that it panics
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic but didn't get one")
					}
				}()
				handler.GetUserIDFromContext(req)
			} else {
				// Get the expected ID from context for comparison
				expectedID := ctx.Value(middleware.UserIDKey).(uuid.UUID)

				// Call the method
				result := handler.GetUserIDFromContext(req)

				// Check result
				if result != expectedID {
					t.Errorf("expected user ID %v, got %v", expectedID, result)
				}
			}
		})
	}
}
