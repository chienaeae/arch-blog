package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/platform/validator"
)

// Theme represents a curated collection of articles
type Theme struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	CuratorID   uuid.UUID // The user who created/manages this theme
	IsActive    bool
	Articles    []*ThemeArticle // Articles in this theme
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Business rule constants
const (
	MaxNameLength        = 100
	MaxSlugLength        = 150
	MaxDescriptionLength = 1000
)

// Validation errors
var (
	ErrInvalidName           = errors.New("name is required and must not exceed 100 characters")
	ErrInvalidSlug           = errors.New("slug is invalid or too long")
	ErrInvalidDescription    = errors.New("description must not exceed 1000 characters")
	ErrInvalidCuratorID      = errors.New("curator ID is required")
	ErrPostNotPublished      = errors.New("only published posts can be added to themes")
	ErrThemeInactive         = errors.New("cannot modify an inactive theme")
	ErrArticleNotFound       = errors.New("article not found in theme")
	ErrInvalidArticleCount   = errors.New("number of post IDs doesn't match number of articles")
	ErrInvalidArticlePostID  = errors.New("post ID not found in theme")
)

// NewTheme creates a new theme with validation
func NewTheme(name, description string, curatorID uuid.UUID) (*Theme, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	// Generate slug from name
	slug := validator.GenerateSlug(name, MaxSlugLength)
	if err := validateThemeSlug(slug); err != nil {
		return nil, err
	}

	if err := validateDescription(description); err != nil {
		return nil, err
	}

	if curatorID == uuid.Nil {
		return nil, ErrInvalidCuratorID
	}

	now := time.Now()
	return &Theme{
		ID:          uuid.New(),
		Name:        name,
		Slug:        slug,
		Description: description,
		CuratorID:   curatorID,
		IsActive:    true,
		Articles:    make([]*ThemeArticle, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update updates the theme details with validation
func (t *Theme) Update(name, description string) error {
	if err := validateName(name); err != nil {
		return err
	}

	if err := validateDescription(description); err != nil {
		return err
	}

	t.Name = name
	t.Description = description
	t.UpdatedAt = time.Now()

	return nil
}

// UpdateSlug updates the theme slug with validation
func (t *Theme) UpdateSlug(slug string) error {
	if err := validateThemeSlug(slug); err != nil {
		return err
	}

	t.Slug = slug
	t.UpdatedAt = time.Now()
	return nil
}

// Deactivate marks the theme as inactive
func (t *Theme) Deactivate() {
	t.IsActive = false
	t.UpdatedAt = time.Now()
}

// Activate marks the theme as active
func (t *Theme) Activate() {
	t.IsActive = true
	t.UpdatedAt = time.Now()
}

// Article Management Methods (Aggregate Root pattern)

// AddArticle adds a post to the theme with business rule validation
func (t *Theme) AddArticle(post PostInfo, addedBy uuid.UUID) error {
	// Business rule: Cannot modify inactive themes
	if !t.IsActive {
		return ErrThemeInactive
	}

	// Business rule: Only published posts can be added to themes
	if !post.IsPublished() {
		return ErrPostNotPublished
	}

	// Check if post is already in the theme
	postID := post.GetID()
	for _, article := range t.Articles {
		if article.PostID == postID {
			return ErrDuplicateArticle
		}
	}

	// Determine the position (add to the end)
	position := len(t.Articles) + 1

	// Create the new article
	article, err := NewThemeArticle(t.ID, postID, position, addedBy)
	if err != nil {
		return err
	}

	// Add to the theme
	t.Articles = append(t.Articles, article)
	t.UpdatedAt = time.Now()

	return nil
}

// RemoveArticle removes a post from the theme
func (t *Theme) RemoveArticle(postID uuid.UUID) error {
	// Business rule: Cannot modify inactive themes
	if !t.IsActive {
		return ErrThemeInactive
	}

	var found bool
	var removedPosition int
	newArticles := make([]*ThemeArticle, 0, len(t.Articles))

	// Remove the article and track its position
	for _, article := range t.Articles {
		if article.PostID == postID {
			found = true
			removedPosition = article.Position
		} else {
			newArticles = append(newArticles, article)
		}
	}

	if !found {
		return ErrArticleNotFound
	}

	// Reposition remaining articles
	for _, article := range newArticles {
		if article.Position > removedPosition {
			article.Position--
			article.UpdatedAt = time.Now()
		}
	}

	t.Articles = newArticles
	t.UpdatedAt = time.Now()

	return nil
}

// ReorderArticles changes the order of articles in the theme
func (t *Theme) ReorderArticles(orderedPostIDs []uuid.UUID) error {
	// Business rule: Cannot modify inactive themes
	if !t.IsActive {
		return ErrThemeInactive
	}

	// Validate that all post IDs are present
	if len(orderedPostIDs) != len(t.Articles) {
		return ErrInvalidArticleCount
	}

	// Create a map of current articles by post ID
	articleMap := make(map[uuid.UUID]*ThemeArticle)
	for _, article := range t.Articles {
		articleMap[article.PostID] = article
	}

	// Validate all post IDs exist in the theme
	for _, postID := range orderedPostIDs {
		if _, exists := articleMap[postID]; !exists {
			return ErrInvalidArticlePostID
		}
	}

	// Update positions based on the new order
	for i, postID := range orderedPostIDs {
		article := articleMap[postID]
		article.Position = i + 1
		article.UpdatedAt = time.Now()
	}

	t.UpdatedAt = time.Now()
	return nil
}

// GetArticle retrieves a specific article from the theme
func (t *Theme) GetArticle(postID uuid.UUID) (*ThemeArticle, bool) {
	for _, article := range t.Articles {
		if article.PostID == postID {
			return article, true
		}
	}
	return nil, false
}

// HasArticle checks if a post is in the theme
func (t *Theme) HasArticle(postID uuid.UUID) bool {
	_, exists := t.GetArticle(postID)
	return exists
}

// ArticleCount returns the number of articles in the theme
func (t *Theme) ArticleCount() int {
	return len(t.Articles)
}

// Validation helpers

func validateName(name string) error {
	if name == "" || len(name) > MaxNameLength {
		return ErrInvalidName
	}
	return nil
}

func validateDescription(description string) error {
	if len(description) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	return nil
}

func validateThemeSlug(slug string) error {
	if err := validator.ValidateSlugFormat(slug, MaxSlugLength); err != nil {
		return ErrInvalidSlug
	}
	return nil
}