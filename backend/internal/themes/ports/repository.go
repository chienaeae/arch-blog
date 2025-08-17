package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/philly/arch-blog/backend/internal/themes/domain"
)

// Repository errors (canonical errors for the repository contract)
var (
	// ErrThemeNotFound is returned when a theme cannot be found
	ErrThemeNotFound = errors.New("theme not found")
	
	// ErrThemeSlugExists is returned when a theme slug already exists
	ErrThemeSlugExists = errors.New("theme slug already exists")
)

// ThemeRepository defines the contract for theme persistence
// Following DDD aggregate pattern: the entire Theme aggregate (including articles) 
// is persisted atomically through a single Save operation
type ThemeRepository interface {
	// Transaction support
	WithTx(tx pgx.Tx) ThemeRepository
	
	// Core aggregate operations
	Create(ctx context.Context, theme *domain.Theme) error
	
	// Save persists the entire aggregate atomically:
	// - Updates theme fields in themes table
	// - Diffs theme.Articles against database state
	// - Performs necessary INSERTs, UPDATEs, and DELETEs on theme_articles
	// All within a single transaction
	Save(ctx context.Context, theme *domain.Theme) error
	
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Loading operations
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Theme, error) // Loads theme without articles
	FindBySlug(ctx context.Context, slug string) (*domain.Theme, error) // Loads theme without articles
	LoadThemeWithArticles(ctx context.Context, id uuid.UUID) (*domain.Theme, error) // Loads full aggregate
	
	// Theme listing and filtering
	ListThemes(ctx context.Context, filter ListFilter) ([]*ThemeSummary, error)
	CountThemes(ctx context.Context, filter ListFilter) (int, error)
	
	// Slug operations
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)
	
	// Theme curator operations (for ownership checks)
	GetThemeCurator(ctx context.Context, themeID uuid.UUID) (uuid.UUID, error)
	ListThemesByCurator(ctx context.Context, curatorID uuid.UUID) ([]*ThemeSummary, error)
}

// ListFilter defines filtering options for theme listings
type ListFilter struct {
	CuratorID *uuid.UUID
	IsActive  *bool
	Limit     int
	Offset    int
}

// ThemeSummary is a lightweight DTO for theme listings
type ThemeSummary struct {
	ID           uuid.UUID
	Name         string
	Slug         string
	Description  string
	CuratorID    uuid.UUID
	CuratorName  string // Joined from users table
	IsActive     bool
	ArticleCount int // Count of articles in the theme
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ArticleDetail provides detailed information about an article in a theme
// Used when loading a theme with its articles for display
type ArticleDetail struct {
	Position    int
	PostID      uuid.UUID
	PostTitle   string
	PostSlug    string
	PostExcerpt string
	AuthorID    uuid.UUID
	AuthorName  string
	AddedBy     uuid.UUID
	AddedByName string
	AddedAt     time.Time
}