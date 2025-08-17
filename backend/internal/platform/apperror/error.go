package apperror

import "fmt"

// AppError is the custom error type for the application.
type AppError struct {
	Code         ErrorCode    // General system-level category (e.g., NOT_FOUND)
	BusinessCode BusinessCode // Specific business reason (e.g., USER_NOT_FOUND)
	Message      string       // Developer-facing message
	HTTPStatus   int          // HTTP status code
	Details      any          // Extra details (e.g., validation errors)
	Inner        error        // Wrapped underlying error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Inner }
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// New creates a new AppError.
func New(code ErrorCode, bizCode BusinessCode, message string, httpStatus int) *AppError {
	return &AppError{Code: code, BusinessCode: bizCode, Message: message, HTTPStatus: httpStatus}
}

// Wrap creates a new AppError that wraps an existing error.
func Wrap(inner error, code ErrorCode, bizCode BusinessCode, message string, httpStatus int) *AppError {
	return &AppError{Code: code, BusinessCode: bizCode, Message: message, HTTPStatus: httpStatus, Inner: inner}
}

// Is allows errors.Is to work with AppError
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	// Match by both Code and BusinessCode for precise matching
	return e.Code == t.Code && e.BusinessCode == t.BusinessCode
}

// Format implements fmt.Formatter for better error output
func (e *AppError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f, "Code: %s, BusinessCode: %s, Message: %s, HTTPStatus: %d",
				e.Code, e.BusinessCode, e.Message, e.HTTPStatus)
			if e.Inner != nil {
				_, _ = fmt.Fprintf(f, "\nCaused by: %+v", e.Inner)
			}
			if e.Details != nil {
				_, _ = fmt.Fprintf(f, "\nDetails: %+v", e.Details)
			}
		} else {
			_, _ = fmt.Fprint(f, e.Message)
		}
	case 's':
		_, _ = fmt.Fprint(f, e.Message)
	}
}
