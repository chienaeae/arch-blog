package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
	"github.com/philly/arch-blog/backend/internal/users/ports"
)

// AuthAdapter bridges the gap between the external authentication provider (Supabase)
// and our internal domain. Its primary responsibility is to take the external
// user ID (from the JWT 'sub' claim provided by the upstream JWT middleware)
// and resolve it to our internal user UUID by querying the database. This allows
// all downstream services and authorization checks to operate with the internal,
// canonical user ID.
//
// NOTE: This middleware introduces a database query into the hot path of EVERY
// authenticated request. While this is a simple and correct approach for now,
// it can become a significant performance bottleneck under high load.
//
// FUTURE OPTIMIZATION: A more performant, long-term solution is to add the
// internal user UUID as a custom claim (e.g., "internal_user_id") to the
// JWT's 'app_metadata'. This can be done via a Supabase Edge Function that is
// triggered on user sign-up. This would eliminate the need for this per-request
// database query and potentially this entire middleware.
type AuthAdapter struct {
	userRepo ports.UserRepository
	logger   logger.Logger
}

// NewAuthAdapter creates a new authentication adapter
func NewAuthAdapter(userRepo ports.UserRepository, logger logger.Logger) *AuthAdapter {
	return &AuthAdapter{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Middleware adapts JWT authentication to work with our authorization system
// It must be placed AFTER JWT middleware and BEFORE authorization middleware
func (a *AuthAdapter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get the subject (Supabase ID) from JWT middleware
		subject, ok := GetJWTUserID(ctx)
		if !ok {
			a.logger.Warn(ctx, "subject not found in context")
			WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Look up the user by Supabase ID to get our internal UUID
		user, err := a.userRepo.FindBySupabaseID(ctx, subject)
		if err != nil {
			a.logger.Error(ctx, "failed to get user by supabase ID",
				"supabase_id", subject,
				"error", err,
			)
			WriteJSONError(w, ErrorCodeNotFound, "User profile not found", http.StatusNotFound)
			return
		}

		// Parse the user ID string to UUID and set it in context for authorization middleware
		userUUID, err := uuid.Parse(user.ID)
		if err != nil {
			a.logger.Error(ctx, "failed to parse user UUID",
				"user_id", user.ID,
				"error", err,
			)
			WriteJSONError(w, ErrorCodeInternalServerError, "Invalid user ID format", http.StatusInternalServerError)
			return
		}
		ctx = SetUserID(ctx, userUUID)

		// Also preserve the email if needed
		if email, ok := GetJWTUserEmail(ctx); ok {
			ctx = context.WithValue(ctx, UserEmailKey, email)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserEmail is a helper to get the user's email from context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}
