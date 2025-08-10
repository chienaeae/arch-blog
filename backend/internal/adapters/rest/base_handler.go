package rest

import (
	"encoding/json"
	"net/http"

	"github.com/philly/arch-blog/backend/internal/adapters/api"
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	error := api.Error{
		Error:   code,
		Message: message,
	}
	
	if err := json.NewEncoder(w).Encode(error); err != nil {
		h.logger.Error(r.Context(), "failed to encode error response", 
			"error", err,
			"error_code", code,
			"status_code", statusCode,
		)
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