package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/adapters/rest/middleware"
	"github.com/philly/arch-blog/backend/internal/platform/apperror"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// BaseHandler contains common dependencies and helper methods for all handlers
type BaseHandler struct {
	logger logger.Logger
}

// NewBaseHandler creates a new base handler with common dependencies
func NewBaseHandler(logger logger.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

// WriteJSONError writes a JSON error response matching OpenAPI spec
func (h *BaseHandler) WriteJSONError(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int) {
	h.writeJSONError(w, r, code, message, statusCode, nil)
}

// writeJSONError is the internal method that supports additional details
func (h *BaseHandler) writeJSONError(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	// Create base error response
	errorResp := map[string]any{
		"error":   code,
		"message": message,
	}
	
	// Add details if provided
	for k, v := range details {
		errorResp[k] = v
	}
	
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Error(r.Context(), "failed to encode error response", 
			"error", err,
			"error_code", code,
			"status_code", statusCode,
		)
	}
}

// HandleError is a generic error handler that translates AppError into JSON responses
func (h *BaseHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperror.AppError
	
	if errors.As(err, &appErr) {
		// The error is a known business error
		details := map[string]any{
			"business_code": string(appErr.BusinessCode),
		}
		if appErr.Details != nil {
			details["context"] = appErr.Details
		}
		
		h.writeJSONError(w, r, string(appErr.Code), appErr.Message, appErr.HTTPStatus, details)
	} else {
		// It's an unexpected error. Log it and return a generic 500 response
		h.logger.Error(r.Context(), "unhandled internal error", "error", err)
		h.writeJSONError(w, r, "INTERNAL_SERVER_ERROR", "An unexpected error occurred", http.StatusInternalServerError, nil)
	}
}

// WriteJSONResponse writes a successful JSON response
func (h *BaseHandler) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error(r.Context(), "failed to encode response", 
			"error", err,
			"status_code", statusCode,
		)
	}
}

// ParseUUID parses a UUID from a string and sends an error response if invalid
func (h *BaseHandler) ParseUUID(w http.ResponseWriter, r *http.Request, value string, paramName string) (uuid.UUID, bool) {
	parsedUUID, err := uuid.Parse(value)
	if err != nil {
		h.WriteJSONError(w, r, "invalid_request", "Invalid "+paramName, http.StatusBadRequest)
		return uuid.Nil, false
	}
	return parsedUUID, true
}

// GetUserIDFromContext is a simple, non-error-handling helper.
// It assumes the middleware has already guaranteed the user ID exists.
func (h *BaseHandler) GetUserIDFromContext(r *http.Request) uuid.UUID {
	// We can ignore the 'ok' because of the middleware contract.
	// If it's not there, it's a programmer error (panic is acceptable),
	// not a user error.
	return r.Context().Value(middleware.UserIDKey).(uuid.UUID)
}