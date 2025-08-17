package apperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/philly/arch-blog/backend/internal/platform/apperror"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		code         apperror.ErrorCode
		businessCode apperror.BusinessCode
		message      string
		httpStatus   int
	}{
		{
			name:         "creates error with all fields",
			code:         apperror.CodeNotFound,
			businessCode: apperror.BusinessCodeUserNotFound,
			message:      "user not found",
			httpStatus:   http.StatusNotFound,
		},
		{
			name:         "creates validation error",
			code:         apperror.CodeValidationFailed,
			businessCode: apperror.BusinessCodeInvalidEmail,
			message:      "invalid email format",
			httpStatus:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apperror.New(tt.code, tt.businessCode, tt.message, tt.httpStatus)

			if err.Code != tt.code {
				t.Errorf("expected code %v, got %v", tt.code, err.Code)
			}
			if err.BusinessCode != tt.businessCode {
				t.Errorf("expected business code %v, got %v", tt.businessCode, err.BusinessCode)
			}
			if err.Message != tt.message {
				t.Errorf("expected message %v, got %v", tt.message, err.Message)
			}
			if err.HTTPStatus != tt.httpStatus {
				t.Errorf("expected HTTP status %v, got %v", tt.httpStatus, err.HTTPStatus)
			}
			if err.Inner != nil {
				t.Errorf("expected no inner error, got %v", err.Inner)
			}
			if err.Details != nil {
				t.Errorf("expected no details, got %v", err.Details)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	innerErr := errors.New("database connection failed")

	err := apperror.Wrap(
		innerErr,
		apperror.CodeInternalError,
		apperror.BusinessCodeGeneral,
		"failed to fetch user",
		http.StatusInternalServerError,
	)

	if err.Inner != innerErr {
		t.Errorf("expected inner error %v, got %v", innerErr, err.Inner)
	}
	if err.Code != apperror.CodeInternalError {
		t.Errorf("expected code %v, got %v", apperror.CodeInternalError, err.Code)
	}
	if err.BusinessCode != apperror.BusinessCodeGeneral {
		t.Errorf("expected business code %v, got %v", apperror.BusinessCodeGeneral, err.BusinessCode)
	}
}

func TestWithDetails(t *testing.T) {
	tests := []struct {
		name    string
		details any
	}{
		{
			name:    "string details",
			details: "additional context",
		},
		{
			name:    "map details",
			details: map[string]string{"field": "email", "reason": "invalid format"},
		},
		{
			name:    "struct details",
			details: struct{ Field string }{Field: "username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apperror.New(
				apperror.CodeValidationFailed,
				apperror.BusinessCodeInvalidFormat,
				"validation failed",
				http.StatusBadRequest,
			)

			errWithDetails := err.WithDetails(tt.details)

			// For maps and structs, we need to compare them differently
			// Just verify the details were set
			if errWithDetails.Details == nil {
				t.Errorf("expected details to be set, but was nil")
			}

			// Verify it returns the same error instance (fluent interface)
			if errWithDetails != err {
				t.Errorf("WithDetails should return the same error instance")
			}
		})
	}
}

func TestError(t *testing.T) {
	message := "test error message"
	err := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		message,
		http.StatusNotFound,
	)

	if err.Error() != message {
		t.Errorf("expected Error() to return %q, got %q", message, err.Error())
	}
}

func TestUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")

	tests := []struct {
		name        string
		err         *apperror.AppError
		expectInner error
	}{
		{
			name: "wrapped error returns inner",
			err: apperror.Wrap(
				innerErr,
				apperror.CodeInternalError,
				apperror.BusinessCodeGeneral,
				"wrapper",
				http.StatusInternalServerError,
			),
			expectInner: innerErr,
		},
		{
			name: "new error returns nil",
			err: apperror.New(
				apperror.CodeNotFound,
				apperror.BusinessCodeUserNotFound,
				"not found",
				http.StatusNotFound,
			),
			expectInner: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.err.Unwrap()
			if unwrapped != tt.expectInner {
				t.Errorf("expected Unwrap() to return %v, got %v", tt.expectInner, unwrapped)
			}
		})
	}
}

func TestIs(t *testing.T) {
	err1 := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"user not found",
		http.StatusNotFound,
	)

	err2 := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"different message",
		http.StatusNotFound,
	)

	err3 := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeRoleNotFound, // Different business code
		"role not found",
		http.StatusNotFound,
	)

	err4 := apperror.New(
		apperror.CodeConflict, // Different error code
		apperror.BusinessCodeUserNotFound,
		"user exists",
		http.StatusConflict,
	)

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error codes match",
			err:    err1,
			target: err2,
			want:   true,
		},
		{
			name:   "different business code doesn't match",
			err:    err1,
			target: err3,
			want:   false,
		},
		{
			name:   "different error code doesn't match",
			err:    err1,
			target: err4,
			want:   false,
		},
		{
			name:   "non-AppError doesn't match",
			err:    err1,
			target: errors.New("regular error"),
			want:   false,
		},
		{
			name:   "errors.Is works with AppError",
			err:    err1,
			target: err1,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	innerErr := errors.New("database error")
	details := map[string]string{"field": "email"}

	err := apperror.Wrap(
		innerErr,
		apperror.CodeValidationFailed,
		apperror.BusinessCodeInvalidEmail,
		"email validation failed",
		http.StatusBadRequest,
	).WithDetails(details)

	tests := []struct {
		name     string
		format   string
		contains []string
	}{
		{
			name:   "simple string format",
			format: "%s",
			contains: []string{
				"email validation failed",
			},
		},
		{
			name:   "simple value format",
			format: "%v",
			contains: []string{
				"email validation failed",
			},
		},
		{
			name:   "verbose format includes all fields",
			format: "%+v",
			contains: []string{
				"Code: VALIDATION_FAILED",
				"BusinessCode: INVALID_EMAIL",
				"Message: email validation failed",
				"HTTPStatus: 400",
				"Caused by: database error",
				"Details: map[field:email]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := fmt.Sprintf(tt.format, err)

			for _, expected := range tt.contains {
				if !contains(output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestFormat_NoInnerError(t *testing.T) {
	err := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"user not found",
		http.StatusNotFound,
	)

	output := fmt.Sprintf("%+v", err)

	if contains(output, "Caused by:") {
		t.Errorf("should not contain 'Caused by:' when there's no inner error, got %q", output)
	}
}

func TestFormat_NoDetails(t *testing.T) {
	err := apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"user not found",
		http.StatusNotFound,
	)

	output := fmt.Sprintf("%+v", err)

	if contains(output, "Details:") {
		t.Errorf("should not contain 'Details:' when there are no details, got %q", output)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
