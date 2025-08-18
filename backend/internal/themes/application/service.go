package application

import (
	"context"
	"errors"
	"net/http"
	"time"

	"backend/internal/platform/apperror"
	"backend/internal/platform/eventbus"
	"backend/internal/platform/events"
	"backend/internal/platform/logger"
	"backend/internal/platform/postgres"
	"backend/internal/platform/validator"
	"backend/internal/themes/domain"
	"backend/internal/themes/ports"
	"github.com/google/uuid"
)

// Error definitions for service operations
var (
	ErrThemeNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeThemeNotFound,
		"theme not found",
		http.StatusNotFound,
	)

	ErrSlugAlreadyExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeSlugAlreadyExists,
		"slug already exists",
		http.StatusConflict,
	)

	ErrInvalidThemeData = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeInvalidFormat,
		"invalid theme data",
		http.StatusBadRequest,
	)

	ErrPostNotPublished = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeCannotAddToTheme,
		"only published posts can be added to themes",
		http.StatusBadRequest,
	)

	ErrThemeInactive = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeInvalidStatusTransition,
		"cannot modify an inactive theme",
		http.StatusConflict,
	)

	ErrPostAlreadyInTheme = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodePostAlreadyInTheme,
		"post is already in this theme",
		http.StatusConflict,
	)

	ErrPostNotInTheme = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodePostNotInTheme,
		"post not found in theme",
		http.StatusNotFound,
	)
)

// PostProvider is an interface to get post information
// This avoids direct dependency on the posts bounded context
type PostProvider interface {
	GetPost(ctx context.Context, id uuid.UUID) (domain.PostInfo, error)
}

// ThemesService handles theme-related business logic
type ThemesService struct {
	txManager    postgres.TransactionManager // Transaction management interface from postgres package
	repo         ports.ThemeRepository
	postProvider PostProvider
	authorizer   ports.Authorizer // Using the port interface
	eventBus     *eventbus.Bus
	logger       logger.Logger
}

// NewThemesService creates a new themes service
func NewThemesService(
	txManager postgres.TransactionManager,
	repo ports.ThemeRepository,
	postProvider PostProvider,
	authorizer ports.Authorizer,
	eventBus *eventbus.Bus,
	logger logger.Logger,
) *ThemesService {
	return &ThemesService{
		txManager:    txManager,
		repo:         repo,
		postProvider: postProvider,
		authorizer:   authorizer,
		eventBus:     eventBus,
		logger:       logger,
	}
}

// CreateThemeParams contains parameters for creating a new theme
type CreateThemeParams struct {
	Name        string
	Description string
}

// CreateTheme creates a new theme
func (s *ThemesService) CreateTheme(ctx context.Context, actorID uuid.UUID, params CreateThemeParams) (*domain.Theme, error) {
	// Check authorization - user must be able to create themes
	canCreate, err := s.authorizer.Can(ctx, actorID, "themes", "create", nil)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canCreate {
		return nil, apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to create themes",
			http.StatusForbidden,
		)
	}
	// Create the theme domain object (it will generate its own slug)
	// The actor becomes the curator
	theme, err := domain.NewTheme(params.Name, params.Description, actorID)
	if err != nil {
		return nil, ErrInvalidThemeData.WithDetails(err.Error())
	}

	// Ensure slug uniqueness
	uniqueSlug, err := s.ensureUniqueSlug(ctx, theme.Slug, nil)
	if err != nil {
		return nil, err
	}

	// Update slug if needed
	if uniqueSlug != theme.Slug {
		if err := theme.UpdateSlug(uniqueSlug); err != nil {
			return nil, ErrInvalidThemeData.WithDetails(err.Error())
		}
	}

	// Save to repository
	if err := s.repo.Create(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to create theme", "error", err)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to create theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeCreatedEvent(ctx, theme, actorID)

	return theme, nil
}

// UpdateThemeParams contains parameters for updating a theme
type UpdateThemeParams struct {
	Name        string
	Description string
}

// UpdateTheme updates an existing theme's details
func (s *ThemesService) UpdateTheme(ctx context.Context, actorID uuid.UUID, id uuid.UUID, params UpdateThemeParams) (*domain.Theme, error) {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return nil, apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	// Load the theme (without articles for performance)
	theme, err := s.getThemeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update the theme details
	if err := theme.Update(params.Name, params.Description); err != nil {
		return nil, ErrInvalidThemeData.WithDetails(err.Error())
	}

	// Check if name changed and we need a new slug
	newSlug := validator.GenerateSlug(params.Name, domain.MaxSlugLength)
	if newSlug != theme.Slug {
		uniqueSlug, err := s.ensureUniqueSlug(ctx, newSlug, &id)
		if err != nil {
			return nil, err
		}
		if err := theme.UpdateSlug(uniqueSlug); err != nil {
			return nil, ErrInvalidThemeData.WithDetails(err.Error())
		}
	}

	// Save to repository (no transaction needed - only updating theme, not articles)
	if err := s.repo.Save(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to update theme", "error", err, "themeID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to update theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeUpdatedEvent(ctx, theme, actorID)

	return theme, nil
}

// AddArticleToTheme adds a post to a theme
func (s *ThemesService) AddArticleToTheme(ctx context.Context, actorID uuid.UUID, themeID, postID uuid.UUID) error {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &themeID)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	// Load the full aggregate with articles
	theme, err := s.repo.LoadThemeWithArticles(ctx, themeID)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to load theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to load theme",
			http.StatusInternalServerError,
		)
	}

	// Get the post information
	post, err := s.postProvider.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	// Add the article using domain logic
	if err := theme.AddArticle(post, actorID); err != nil {
		// Map domain errors to service errors
		switch {
		case errors.Is(err, domain.ErrPostNotPublished):
			return ErrPostNotPublished
		case errors.Is(err, domain.ErrThemeInactive):
			return ErrThemeInactive
		case errors.Is(err, domain.ErrDuplicateArticle):
			return ErrPostAlreadyInTheme
		default:
			return ErrInvalidThemeData.WithDetails(err.Error())
		}
	}

	// Save the entire aggregate atomically within a transaction
	if err := s.saveThemeWithTransaction(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to save theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to add article to theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	// Find the position of the newly added article
	if article, exists := theme.GetArticle(postID); exists {
		s.publishThemeArticleAddedEvent(ctx, themeID, postID, article.Position, actorID)
	}

	return nil
}

// RemoveArticleFromTheme removes a post from a theme
func (s *ThemesService) RemoveArticleFromTheme(ctx context.Context, actorID uuid.UUID, themeID, postID uuid.UUID) error {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &themeID)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	// Load the full aggregate with articles
	theme, err := s.repo.LoadThemeWithArticles(ctx, themeID)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to load theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to load theme",
			http.StatusInternalServerError,
		)
	}

	// Remove the article using domain logic
	if err := theme.RemoveArticle(postID); err != nil {
		// Map domain errors to service errors
		switch {
		case errors.Is(err, domain.ErrThemeInactive):
			return ErrThemeInactive
		case errors.Is(err, domain.ErrArticleNotFound):
			return ErrPostNotInTheme
		default:
			return ErrInvalidThemeData.WithDetails(err.Error())
		}
	}

	// Save the entire aggregate atomically within a transaction
	if err := s.saveThemeWithTransaction(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to save theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to remove article from theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeArticleRemovedEvent(ctx, themeID, postID, actorID)

	return nil
}

// ReorderThemeArticles changes the order of articles in a theme
func (s *ThemesService) ReorderThemeArticles(ctx context.Context, actorID uuid.UUID, themeID uuid.UUID, orderedPostIDs []uuid.UUID) error {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &themeID)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	// Load the full aggregate with articles
	theme, err := s.repo.LoadThemeWithArticles(ctx, themeID)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to load theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to load theme",
			http.StatusInternalServerError,
		)
	}

	// Reorder articles using domain logic
	if err := theme.ReorderArticles(orderedPostIDs); err != nil {
		// Map domain errors to service errors
		switch {
		case errors.Is(err, domain.ErrThemeInactive):
			return ErrThemeInactive
		case errors.Is(err, domain.ErrInvalidArticleCount):
			return apperror.New(
				apperror.CodeValidationFailed,
				apperror.BusinessCodeInvalidFormat,
				err.Error(),
				http.StatusBadRequest,
			)
		case errors.Is(err, domain.ErrInvalidArticlePostID):
			return ErrPostNotInTheme
		default:
			return ErrInvalidThemeData.WithDetails(err.Error())
		}
	}

	// Save the entire aggregate atomically within a transaction
	if err := s.saveThemeWithTransaction(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to save theme", "error", err, "themeID", themeID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to reorder theme articles",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeArticlesReorderedEvent(ctx, themeID, orderedPostIDs, actorID)

	return nil
}

// ActivateTheme activates an inactive theme
func (s *ThemesService) ActivateTheme(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	theme, err := s.getThemeByID(ctx, id)
	if err != nil {
		return err
	}

	theme.Activate()

	if err := s.repo.Save(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to activate theme", "error", err, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to activate theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeActivatedEvent(ctx, theme, actorID)

	return nil
}

// DeactivateTheme deactivates an active theme
func (s *ThemesService) DeactivateTheme(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	// Check authorization - user must be able to update this specific theme
	canUpdate, err := s.authorizer.Can(ctx, actorID, "themes", "update", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUpdate {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to update this theme",
			http.StatusForbidden,
		)
	}
	theme, err := s.getThemeByID(ctx, id)
	if err != nil {
		return err
	}

	theme.Deactivate()

	if err := s.repo.Save(ctx, theme); err != nil {
		s.logger.Error(ctx, "failed to deactivate theme", "error", err, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to deactivate theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeDeactivatedEvent(ctx, theme, actorID)

	return nil
}

// DeleteTheme removes a theme from the system
func (s *ThemesService) DeleteTheme(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	// Check authorization - user must be able to delete this specific theme
	canDelete, err := s.authorizer.Can(ctx, actorID, "themes", "delete", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canDelete {
		return apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to delete this theme",
			http.StatusForbidden,
		)
	}
	// Check if theme exists
	_, err = s.getThemeByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error(ctx, "failed to delete theme", "error", err, "themeID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to delete theme",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishThemeDeletedEvent(ctx, id, actorID)

	return nil
}

// GetTheme retrieves a theme by ID (without articles)
func (s *ThemesService) GetTheme(ctx context.Context, id uuid.UUID) (*domain.Theme, error) {
	return s.getThemeByID(ctx, id)
}

// GetThemeBySlug retrieves a theme by its slug (without articles)
func (s *ThemesService) GetThemeBySlug(ctx context.Context, slug string) (*domain.Theme, error) {
	theme, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return nil, ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to find theme by slug", "error", err, "slug", slug)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to retrieve theme",
			http.StatusInternalServerError,
		)
	}
	return theme, nil
}

// GetThemeWithArticles retrieves a theme with all its articles
func (s *ThemesService) GetThemeWithArticles(ctx context.Context, id uuid.UUID) (*domain.Theme, error) {
	theme, err := s.repo.LoadThemeWithArticles(ctx, id)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return nil, ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to load theme with articles", "error", err, "themeID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to retrieve theme",
			http.StatusInternalServerError,
		)
	}
	return theme, nil
}

// ListThemes retrieves a list of theme summaries
func (s *ThemesService) ListThemes(ctx context.Context, filter ports.ListFilter) ([]*ports.ThemeSummary, int, error) {
	summaries, err := s.repo.ListThemes(ctx, filter)
	if err != nil {
		s.logger.Error(ctx, "failed to list themes", "error", err)
		return nil, 0, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to list themes",
			http.StatusInternalServerError,
		)
	}

	count, err := s.repo.CountThemes(ctx, filter)
	if err != nil {
		s.logger.Error(ctx, "failed to count themes", "error", err)
		return nil, 0, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to count themes",
			http.StatusInternalServerError,
		)
	}

	return summaries, count, nil
}

// Private helper methods

// getThemeByID fetches a theme and handles not-found errors consistently
func (s *ThemesService) getThemeByID(ctx context.Context, id uuid.UUID) (*domain.Theme, error) {
	theme, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ports.ErrThemeNotFound) {
			return nil, ErrThemeNotFound
		}
		s.logger.Error(ctx, "failed to find theme", "error", err, "themeID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to retrieve theme",
			http.StatusInternalServerError,
		)
	}
	return theme, nil
}

// saveThemeWithTransaction saves a theme within a transaction
// This is used when saving a theme with articles to ensure atomicity
func (s *ThemesService) saveThemeWithTransaction(ctx context.Context, theme *domain.Theme) error {
	// Begin transaction
	tx, err := s.txManager.BeginTx(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed to begin transaction", "error", err, "themeID", theme.ID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to begin transaction",
			http.StatusInternalServerError,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Get a transactional repository
	txRepo := s.repo.WithTx(tx.Tx())

	// Save the theme using the transactional repository
	if err := txRepo.Save(ctx, theme); err != nil {
		// The error is already logged at repository level
		// Just return it as-is (repository errors should already be apperror types)
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		s.logger.Error(ctx, "failed to commit transaction", "error", err, "themeID", theme.ID)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to commit transaction",
			http.StatusInternalServerError,
		)
	}

	return nil
}

// ensureUniqueSlug ensures a slug is unique, potentially adding a numeric suffix
func (s *ThemesService) ensureUniqueSlug(ctx context.Context, baseSlug string, excludeID *uuid.UUID) (string, error) {
	slug := baseSlug
	suffix := 1

	for {
		exists, err := s.repo.SlugExists(ctx, slug, excludeID)
		if err != nil {
			s.logger.Error(ctx, "failed to check slug existence", "error", err, "slug", slug)
			return "", apperror.New(
				apperror.CodeInternalError,
				apperror.BusinessCodeGeneral,
				"failed to validate slug",
				http.StatusInternalServerError,
			)
		}

		if !exists {
			return slug, nil
		}

		// Try with a suffix
		slug = validator.MakeSlugUniqueWithMaxLength(baseSlug, suffix, domain.MaxSlugLength)
		suffix++

		// Prevent infinite loop
		if suffix > 100 {
			return "", ErrSlugAlreadyExists.WithDetails("unable to generate unique slug")
		}
	}
}

// Event publishing methods

func (s *ThemesService) publishThemeCreatedEvent(ctx context.Context, theme *domain.Theme, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeCreatedTopic,
		Payload: events.ThemeCreatedEvent{
			ThemeID:    theme.ID,
			ActorID:    actorID,
			Name:       theme.Name,
			Slug:       theme.Slug,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeUpdatedEvent(ctx context.Context, theme *domain.Theme, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeUpdatedTopic,
		Payload: events.ThemeUpdatedEvent{
			ThemeID:    theme.ID,
			ActorID:    actorID,
			Name:       theme.Name,
			Slug:       theme.Slug,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeActivatedEvent(ctx context.Context, theme *domain.Theme, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeActivatedTopic,
		Payload: events.ThemeActivatedEvent{
			ThemeID:    theme.ID,
			ActorID:    actorID,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeDeactivatedEvent(ctx context.Context, theme *domain.Theme, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeDeactivatedTopic,
		Payload: events.ThemeDeactivatedEvent{
			ThemeID:    theme.ID,
			ActorID:    actorID,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeDeletedEvent(ctx context.Context, themeID uuid.UUID, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeDeletedTopic,
		Payload: events.ThemeDeletedEvent{
			ThemeID:    themeID,
			ActorID:    actorID,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeArticleAddedEvent(ctx context.Context, themeID, postID uuid.UUID, position int, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeArticleAddedTopic,
		Payload: events.ThemeArticleAddedEvent{
			ThemeID:    themeID,
			PostID:     postID,
			Position:   position,
			ActorID:    actorID,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeArticleRemovedEvent(ctx context.Context, themeID, postID, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeArticleRemovedTopic,
		Payload: events.ThemeArticleRemovedEvent{
			ThemeID:    themeID,
			PostID:     postID,
			ActorID:    actorID,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *ThemesService) publishThemeArticlesReorderedEvent(ctx context.Context, themeID uuid.UUID, orderedPostIDs []uuid.UUID, actorID uuid.UUID) {
	event := eventbus.Event{
		Topic: events.ThemeArticlesReorderedTopic,
		Payload: events.ThemeArticlesReorderedEvent{
			ThemeID:        themeID,
			OrderedPostIDs: orderedPostIDs,
			ActorID:        actorID,
			OccurredAt:     time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}
