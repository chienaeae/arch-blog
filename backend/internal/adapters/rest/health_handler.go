package rest

import (
	"context"
	"net/http"
	"time"

	"backend/internal/adapters/api"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker provides methods to check system health
type HealthChecker interface {
	CheckDatabase(ctx context.Context) error
	// Add more checks as needed (Redis, external services, etc.)
}

type HealthHandler struct {
	*BaseHandler
	version string
	pool    *pgxpool.Pool // For readiness check
}

func NewHealthHandler(base *BaseHandler, version string, pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		BaseHandler: base,
		version:     version,
		pool:        pool,
	}
}

// GetLiveness implements the liveness probe endpoint
// This is a lightweight check with no external dependencies
func (h *HealthHandler) GetLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple check - if we can respond, we're alive
	status := api.Healthy
	timestamp := time.Now()

	response := api.HealthStatus{
		Status:    status,
		Timestamp: timestamp,
		Version:   &h.version,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetReadiness implements the readiness probe endpoint
// This checks all critical dependencies
func (h *HealthHandler) GetReadiness(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now()
	status := api.Healthy
	httpStatus := http.StatusOK

	// Create the checks struct
	var checks *struct {
		Database *api.HealthStatusChecksDatabase `json:"database,omitempty"`
	}

	// Check database connectivity
	if h.pool != nil {
		checks = &struct {
			Database *api.HealthStatusChecksDatabase `json:"database,omitempty"`
		}{}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.pool.Ping(ctx); err != nil {
			dbStatus := api.Down
			checks.Database = &dbStatus
			status = api.Unhealthy
			httpStatus = http.StatusServiceUnavailable
		} else {
			dbStatus := api.Up
			checks.Database = &dbStatus
		}
	} else {
		status = api.Degraded
	}

	response := api.HealthStatus{
		Status:    status,
		Timestamp: timestamp,
		Version:   &h.version,
		Checks:    checks,
	}

	h.WriteJSONResponse(w, r, response, httpStatus)
}
