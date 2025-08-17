package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/platform/validator"
)

// PostStatus represents the publication state of a post
type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusArchived  PostStatus = "archived"
)

// IsValid checks if the status is a valid value
func (s PostStatus) IsValid() bool {
	switch s {
	case PostStatusDraft, PostStatusPublished, PostStatusArchived:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a status transition is allowed
func (s PostStatus) CanTransitionTo(target PostStatus) bool {
	switch s {
	case PostStatusDraft:
		// Draft can go to published or archived
		return target == PostStatusPublished || target == PostStatusArchived
	case PostStatusPublished:
		// Published can only go to archived
		return target == PostStatusArchived
	case PostStatusArchived:
		// Archived can go back to draft or published
		return target == PostStatusDraft || target == PostStatusPublished
	default:
		return false
	}
}

// Post represents a blog post in the domain
type Post struct {
	ID          uuid.UUID
	Title       string
	Slug        string
	Content     string // HTML content
	Excerpt     string // Plain text excerpt
	AuthorID    uuid.UUID
	Status      PostStatus
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Business rule constants
const (
	MaxTitleLength   = 200
	MaxSlugLength    = 250
	MaxExcerptLength = 500
)

// Validation errors
var (
	ErrInvalidTitle      = errors.New("title is required and must not exceed 200 characters")
	ErrInvalidSlug       = errors.New("slug is invalid or too long")
	ErrInvalidContent    = errors.New("content is required")
	ErrInvalidExcerpt    = errors.New("excerpt must not exceed 500 characters")
	ErrInvalidAuthorID   = errors.New("author ID is required")
	ErrInvalidStatus     = errors.New("invalid post status")
	ErrInvalidTransition = errors.New("invalid status transition")
)


// NewPost creates a new post with validation
func NewPost(title, content, excerpt string, authorID uuid.UUID) (*Post, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}

	// Generate slug from title
	slug := validator.GenerateSlug(title, MaxSlugLength)
	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	if err := validateContent(content); err != nil {
		return nil, err
	}

	if err := validateExcerpt(excerpt); err != nil {
		return nil, err
	}

	if authorID == uuid.Nil {
		return nil, ErrInvalidAuthorID
	}

	now := time.Now()
	return &Post{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		Excerpt:    excerpt,
		AuthorID:   authorID,
		Status:     PostStatusDraft,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdateContent updates the post content with validation
func (p *Post) UpdateContent(title, content, excerpt string) error {
	if err := validateTitle(title); err != nil {
		return err
	}

	if err := validateContent(content); err != nil {
		return err
	}

	if err := validateExcerpt(excerpt); err != nil {
		return err
	}

	p.Title = title
	p.Content = content
	p.Excerpt = excerpt
	p.UpdatedAt = time.Now()

	return nil
}

// UpdateSlug updates the post slug with validation
// Note: Slug uniqueness must be checked by the service layer before calling this
func (p *Post) UpdateSlug(slug string) error {
	if err := validateSlug(slug); err != nil {
		return err
	}

	p.Slug = slug
	p.UpdatedAt = time.Now()
	return nil
}

// Publish transitions the post to published status
func (p *Post) Publish() error {
	if !p.Status.CanTransitionTo(PostStatusPublished) {
		return fmt.Errorf("%w: cannot publish from %s", ErrInvalidTransition, p.Status)
	}

	p.Status = PostStatusPublished
	now := time.Now()
	p.PublishedAt = &now
	p.UpdatedAt = now
	return nil
}

// Archive transitions the post to archived status
func (p *Post) Archive() error {
	if !p.Status.CanTransitionTo(PostStatusArchived) {
		return fmt.Errorf("%w: cannot archive from %s", ErrInvalidTransition, p.Status)
	}

	p.Status = PostStatusArchived
	p.UpdatedAt = time.Now()
	return nil
}

// Unpublish transitions the post back to draft status
func (p *Post) Unpublish() error {
	if !p.Status.CanTransitionTo(PostStatusDraft) {
		return fmt.Errorf("%w: cannot unpublish from %s", ErrInvalidTransition, p.Status)
	}

	p.Status = PostStatusDraft
	p.PublishedAt = nil
	p.UpdatedAt = time.Now()
	return nil
}

// IsPublished checks if the post is currently published
func (p *Post) IsPublished() bool {
	return p.Status == PostStatusPublished
}

// CanBeAddedToTheme checks if the post can be added to a theme
func (p *Post) CanBeAddedToTheme() bool {
	return p.Status == PostStatusPublished
}

// GetID returns the post ID
// Implements themes/domain.PostInfo interface
func (p *Post) GetID() uuid.UUID {
	return p.ID
}

// GetAuthorID returns the post author ID
// Implements themes/domain.PostInfo interface
func (p *Post) GetAuthorID() uuid.UUID {
	return p.AuthorID
}


// Validation helpers

func validateTitle(title string) error {
	if title == "" || len(title) > MaxTitleLength {
		return ErrInvalidTitle
	}
	return nil
}

func validateContent(content string) error {
	if content == "" {
		return ErrInvalidContent
	}
	// Additional HTML sanitization would be done at the application layer
	return nil
}

func validateExcerpt(excerpt string) error {
	if len(excerpt) > MaxExcerptLength {
		return ErrInvalidExcerpt
	}
	return nil
}

func validateSlug(slug string) error {
	if err := validator.ValidateSlugFormat(slug, MaxSlugLength); err != nil {
		return ErrInvalidSlug
	}
	return nil
}