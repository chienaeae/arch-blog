package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ThemeArticle represents a post included in a theme
// This is part of the Theme aggregate and should only be created/modified through Theme methods
type ThemeArticle struct {
	ID        uuid.UUID
	ThemeID   uuid.UUID
	PostID    uuid.UUID
	Position  int // Order within the theme (1-based)
	AddedBy   uuid.UUID
	AddedAt   time.Time
	UpdatedAt time.Time
}

// Additional validation errors for articles
var (
	ErrDuplicateArticle = errors.New("post is already in this theme")
)

// NewThemeArticle creates a new theme article association
// This is an internal factory used by the Theme aggregate
func NewThemeArticle(themeID, postID uuid.UUID, position int, addedBy uuid.UUID) (*ThemeArticle, error) {
	if themeID == uuid.Nil {
		return nil, errors.New("theme ID is required")
	}

	if postID == uuid.Nil {
		return nil, errors.New("post ID is required")
	}

	if position <= 0 {
		return nil, errors.New("position must be greater than 0")
	}

	if addedBy == uuid.Nil {
		return nil, errors.New("added by user ID is required")
	}

	now := time.Now()
	return &ThemeArticle{
		ID:        uuid.New(),
		ThemeID:   themeID,
		PostID:    postID,
		Position:  position,
		AddedBy:   addedBy,
		AddedAt:   now,
		UpdatedAt: now,
	}, nil
}