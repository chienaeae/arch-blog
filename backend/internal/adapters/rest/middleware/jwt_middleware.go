package middleware

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
	ErrMissingToken   = errors.New("missing authentication token")
	ErrInvalidToken   = errors.New("invalid authentication token")
	ErrTokenExpired   = errors.New("token has expired")
	ErrInvalidIssuer  = errors.New("invalid token issuer")
	ErrMissingSubject = errors.New("missing subject in token")
	ErrMissingEmail   = errors.New("missing email in token")
)

type jwtContextKey string

const (
	JWTUserIDContextKey    jwtContextKey = "jwt_user_id"
	JWTUserEmailContextKey jwtContextKey = "jwt_email"
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
			WriteJSONError(w, ErrorCodeUnauthorized, ErrMissingToken.Error(), http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			WriteJSONError(w, ErrorCodeUnauthorized, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Get the cached key set
		keySet, err := m.cache.Lookup(r.Context(), m.jwksEndpoint)
		if err != nil {
			WriteJSONError(w, ErrorCodeInternalServerError, fmt.Sprintf("Failed to get JWKS: %v", err), http.StatusInternalServerError)
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
				WriteJSONError(w, ErrorCodeTokenExpired, ErrTokenExpired.Error(), http.StatusUnauthorized)
				return
			}
			WriteJSONError(w, ErrorCodeInvalidToken, ErrInvalidToken.Error(), http.StatusUnauthorized)
			return
		}

		// Extract required claims
		var subject string
		err = token.Get("sub", &subject)
		if err != nil {
			WriteJSONError(w, ErrorCodeInvalidToken, ErrMissingSubject.Error(), http.StatusUnauthorized)
			return
		}

		var email string
		err = token.Get("email", &email)
		if err != nil {
			WriteJSONError(w, ErrorCodeInvalidToken, ErrMissingEmail.Error(), http.StatusUnauthorized)
			return
		}

		// Convert to strings
		if subject == "" {
			WriteJSONError(w, ErrorCodeInvalidToken, "Invalid subject format", http.StatusUnauthorized)
			return
		}

		if email == "" {
			WriteJSONError(w, ErrorCodeInvalidToken, "Invalid email format", http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), JWTUserIDContextKey, subject)
		ctx = context.WithValue(ctx, JWTUserEmailContextKey, email)

		// Continue with the request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetJWTUserID extracts the user ID from the request context set by JWT middleware
func GetJWTUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(JWTUserIDContextKey).(string)
	return userID, ok
}

// GetJWTUserEmail extracts the user email from the request context set by JWT middleware
func GetJWTUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(JWTUserEmailContextKey).(string)
	return email, ok
}
