package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/posts/domain"
)

// Repository errors - these are the canonical errors that repository
// implementations should return. The PostgreSQL implementation will
// translate pgx.ErrNoRows to these errors.
var (
	// ErrPostNotFound is returned when a post cannot be found
	ErrPostNotFound = errors.New("post not found")
)

// PostSummary is a lightweight DTO for list views
// It contains only the essential fields needed for displaying posts in lists
type PostSummary struct {
	ID          uuid.UUID
	Title       string
	Slug        string
	Excerpt     string
	AuthorID    uuid.UUID
	AuthorName  string // Joined from users table
	Status      domain.PostStatus
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PostRepository defines the interface for post persistence
type PostRepository interface {
	// Create saves a new post to the database
	Create(ctx context.Context, post *domain.Post) error

	// FindByID retrieves a full post by its ID (includes content)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error)

	// FindBySlug retrieves a full post by its slug (includes content)
	FindBySlug(ctx context.Context, slug string) (*domain.Post, error)

	// Update modifies an existing post
	Update(ctx context.Context, post *domain.Post) error

	// Delete removes a post from the database
	Delete(ctx context.Context, id uuid.UUID) error

	// ListSummaries retrieves post summaries for efficient list views
	// Returns summaries without the heavy content field
	ListSummaries(ctx context.Context, filter ListFilter) ([]*PostSummary, error)

	// Count returns the total number of posts matching the filter
	Count(ctx context.Context, filter ListFilter) (int, error)

	// SlugExists checks if a slug is already in use
	// Optionally excludes a specific post ID (for updates)
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// FindSummariesByAuthor retrieves post summaries by a specific author
	FindSummariesByAuthor(ctx context.Context, authorID uuid.UUID, filter ListFilter) ([]*PostSummary, error)

	// GetPostAuthor retrieves just the author ID for a post (for ownership checks)
	GetPostAuthor(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)
}

// ListFilter contains filtering and pagination options for listing posts
type ListFilter struct {
	// Status filters by post status (nil means all statuses)
	Status *domain.PostStatus

	// AuthorID filters by author (nil means all authors)
	AuthorID *uuid.UUID

	// SearchQuery for full-text search in title and excerpt
	SearchQuery string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	OrderBy   OrderField
	OrderDesc bool
}

// OrderField represents the field to order posts by
type OrderField string

const (
	OrderByCreatedAt   OrderField = "created_at"
	OrderByUpdatedAt   OrderField = "updated_at"
	OrderByPublishedAt OrderField = "published_at"
	OrderByTitle       OrderField = "title"
)

// DefaultListFilter returns a sensible default filter
func DefaultListFilter() ListFilter {
	return ListFilter{
		Limit:     20,
		Offset:    0,
		OrderBy:   OrderByCreatedAt,
		OrderDesc: true,
	}
}