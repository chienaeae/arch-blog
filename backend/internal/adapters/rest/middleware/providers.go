package middleware

import (
	"context"

	"github.com/google/wire"
	authzApp "github.com/philly/arch-blog/backend/internal/authz/application"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
	"github.com/philly/arch-blog/backend/internal/users/ports"
)

// ProviderSet is the wire provider set for middleware components
var ProviderSet = wire.NewSet(
	ProvideJWTMiddleware,
	ProvideAuthAdapter,
	ProvideAuthorizationMiddleware,
)

// JWTConfig carries the minimal settings needed to construct the JWT middleware
type JWTConfig struct {
	JWKS   string
	Issuer string
}

// ProvideJWTMiddleware creates JWT middleware from JWTConfig
func ProvideJWTMiddleware(ctx context.Context, cfg JWTConfig) (*JWTMiddleware, error) {
	return NewJWTMiddleware(ctx, cfg.JWKS, cfg.Issuer)
}

// ProvideAuthAdapter creates the auth adapter middleware
func ProvideAuthAdapter(userRepo ports.UserRepository, log logger.Logger) *AuthAdapter {
	return NewAuthAdapter(userRepo, log)
}

// ProvideAuthorizationMiddleware creates the authorization middleware
func ProvideAuthorizationMiddleware(authzService *authzApp.AuthzService, log logger.Logger) *AuthorizationMiddleware {
	return NewAuthorizationMiddleware(authzService, log)
}
