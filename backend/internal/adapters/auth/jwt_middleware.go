package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

var (
	ErrMissingToken     = errors.New("missing authentication token")
	ErrInvalidToken     = errors.New("invalid authentication token")
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidIssuer    = errors.New("invalid token issuer")
	ErrMissingSubject   = errors.New("missing subject in token")
	ErrMissingEmail     = errors.New("missing email in token")
)

type contextKey string

const (
	UserIDContextKey    contextKey = "user_id"
	UserEmailContextKey contextKey = "email"
)

type JWTMiddleware struct {
	jwksEndpoint string
	issuer       string
	cache        *jwk.Cache
}

func NewJWTMiddleware(ctx context.Context, jwksEndpoint string, issuer string) (*JWTMiddleware, error) {

	// Create a cache with automatic refresh
	cache, err := jwk.NewCache(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	// Configure the cache to refresh every 15 minutes
	if err := cache.Register(ctx, jwksEndpoint); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	// Perform initial fetch to validate the URL
	_, err = cache.Lookup(ctx, jwksEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial JWKS: %w", err)
	}

	return &JWTMiddleware{
		jwksEndpoint: jwksEndpoint,
		issuer:       issuer,
		cache:        cache,
	}, nil
}

func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, "unauthorized", ErrMissingToken.Error(), http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeError(w, "unauthorized", "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Get the cached key set
		keySet, err := m.cache.Lookup(r.Context(), m.jwksEndpoint)
		if err != nil {
			writeError(w, "internal_server_error", fmt.Sprintf("Failed to get JWKS: %v", err), http.StatusInternalServerError)
			return
		}

		// Parse and validate the token
		token, err := jwt.ParseString(
			tokenString,
			jwt.WithKeySet(keySet),
			jwt.WithValidate(true),
			jwt.WithIssuer(m.issuer),
		)
		if err != nil {
			// Check if token is expired
			if err.Error() == "exp not satisfied" || strings.Contains(err.Error(), "expired") {
				writeError(w, "token_expired", ErrTokenExpired.Error(), http.StatusUnauthorized)
				return
			}
			writeError(w, "invalid_token", ErrInvalidToken.Error(), http.StatusUnauthorized)
			return
		}

		// Extract required claims
		var subject string
		err = token.Get("sub", &subject)
		if err != nil {
			writeError(w, "invalid_token", ErrMissingSubject.Error(), http.StatusUnauthorized)
			return
		}

		var email string
		err = token.Get("email", &email)
		if err != nil {
			writeError(w, "invalid_token", ErrMissingEmail.Error(), http.StatusUnauthorized)
			return
		}

		// Convert to strings
		if subject == "" {
			writeError(w, "invalid_token", "Invalid subject format", http.StatusUnauthorized)
			return
		}

		if email == "" {
			writeError(w, "invalid_token", "Invalid email format", http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDContextKey, subject)
		ctx = context.WithValue(ctx, UserEmailContextKey, email)

		// Continue with the request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeError writes a JSON error response that matches our OpenAPI specification
func writeError(w http.ResponseWriter, code string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, `{"error": "%s", "message": "%s"}`, code, message)
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}

// GetUserEmail extracts the user email from the request context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailContextKey).(string)
	return email, ok
}