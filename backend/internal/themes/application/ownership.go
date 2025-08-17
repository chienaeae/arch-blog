package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
	"github.com/philly/arch-blog/backend/internal/platform/ownership"
	"github.com/philly/arch-blog/backend/internal/themes/ports"
)

// ThemesOwnershipChecker checks ownership of themes
// It depends directly on the repository, not the service, for cleaner architecture
type ThemesOwnershipChecker struct {
	repo   ports.ThemeRepository
	logger logger.Logger
}

// NewThemesOwnershipChecker creates a new themes ownership checker
func NewThemesOwnershipChecker(repo ports.ThemeRepository, logger logger.Logger) *ThemesOwnershipChecker {
	return &ThemesOwnershipChecker{
		repo:   repo,
		logger: logger,
	}
}

// CheckOwnership checks if a user owns (curates) a specific theme
// Implements the ownership.Checker interface
func (t *ThemesOwnershipChecker) CheckOwnership(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (bool, error) {
	curatorID, err := t.repo.GetThemeCurator(ctx, resourceID)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			// Theme doesn't exist, so user doesn't own it
			return false, nil
		}
		t.logger.Error(ctx, "failed to get theme curator", "error", err, "themeID", resourceID)
		return false, err
	}
	
	return curatorID == userID, nil
}

// RegisterThemesOwnership registers the themes ownership checker with the registry
func RegisterThemesOwnership(registry ownership.Registry, repo ports.ThemeRepository, logger logger.Logger) {
	checker := NewThemesOwnershipChecker(repo, logger)
	registry.RegisterChecker("themes", checker)
}