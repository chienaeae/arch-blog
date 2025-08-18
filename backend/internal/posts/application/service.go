package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"backend/internal/platform/apperror"
	"backend/internal/platform/eventbus"
	"backend/internal/platform/events"
	"backend/internal/platform/logger"
	"backend/internal/platform/validator"
	"backend/internal/posts/domain"
	"backend/internal/posts/ports"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
)

// Error definitions for service operations
var (
	ErrPostNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodePostNotFound,
		"post not found",
		http.StatusNotFound,
	)

	ErrSlugAlreadyExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeSlugAlreadyExists,
		"slug already exists",
		http.StatusConflict,
	)

	ErrInvalidStatusTransition = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeInvalidStatusTransition,
		"invalid status transition",
		http.StatusConflict,
	)

	ErrInvalidPostData = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeInvalidFormat,
		"invalid post data",
		http.StatusBadRequest,
	)
)

// PostsService handles post-related business logic
type PostsService struct {
	repo       ports.PostRepository
	authorizer ports.Authorizer
	eventBus   *eventbus.Bus
	logger     logger.Logger
	sanitizer  *bluemonday.Policy
}

// NewPostsService creates a new posts service
func NewPostsService(
	repo ports.PostRepository,
	authorizer ports.Authorizer,
	eventBus *eventbus.Bus,
	logger logger.Logger,
) *PostsService {
	// Create a strict HTML sanitizer policy
	sanitizer := bluemonday.UGCPolicy()

	return &PostsService{
		repo:       repo,
		authorizer: authorizer,
		eventBus:   eventBus,
		logger:     logger,
		sanitizer:  sanitizer,
	}
}

// CreatePostParams contains parameters for creating a new post
type CreatePostParams struct {
	Title   string
	Content string
	Excerpt string
}

// CreatePost creates a new blog post
func (s *PostsService) CreatePost(ctx context.Context, actorID uuid.UUID, params CreatePostParams) (*domain.Post, error) {
	// Check authorization - user must be able to create posts
	canCreate, err := s.authorizer.Can(ctx, actorID, "posts", "create", nil)
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
			"not authorized to create posts",
			http.StatusForbidden,
		)
	}
	// Sanitize HTML content
	sanitizedContent := s.sanitizer.Sanitize(params.Content)

	// Create the post domain object (it will generate its own slug)
	// The actor becomes the author
	post, err := domain.NewPost(
		params.Title,
		sanitizedContent,
		params.Excerpt,
		actorID,
	)
	if err != nil {
		return nil, ErrInvalidPostData.WithDetails(err.Error())
	}

	// Ensure slug uniqueness
	uniqueSlug, err := s.ensureUniqueSlug(ctx, post.Slug, nil)
	if err != nil {
		return nil, err
	}

	// Update slug if needed
	if uniqueSlug != post.Slug {
		if err := post.UpdateSlug(uniqueSlug); err != nil {
			return nil, ErrInvalidPostData.WithDetails(err.Error())
		}
	}

	// Save to repository
	if err := s.repo.Create(ctx, post); err != nil {
		s.logger.Error(ctx, "failed to create post", "error", err)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to create post",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishPostCreatedEvent(ctx, post)

	return post, nil
}

// UpdatePostParams contains parameters for updating a post
type UpdatePostParams struct {
	Title   string
	Content string
	Excerpt string
}

// UpdatePost updates an existing post
func (s *PostsService) UpdatePost(ctx context.Context, actorID uuid.UUID, id uuid.UUID, params UpdatePostParams) (*domain.Post, error) {
	// Check authorization - user must be able to update this specific post
	canUpdate, err := s.authorizer.Can(ctx, actorID, "posts", "update", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "postID", id)
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
			"not authorized to update this post",
			http.StatusForbidden,
		)
	}
	// Fetch the existing post
	post, err := s.getPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Sanitize HTML content
	sanitizedContent := s.sanitizer.Sanitize(params.Content)

	// Update the post content
	if err := post.UpdateContent(params.Title, sanitizedContent, params.Excerpt); err != nil {
		return nil, ErrInvalidPostData.WithDetails(err.Error())
	}

	// Check if title changed and we need a new slug
	newSlug := validator.GenerateSlug(params.Title, domain.MaxSlugLength)
	if newSlug != post.Slug {
		uniqueSlug, err := s.ensureUniqueSlug(ctx, newSlug, &id)
		if err != nil {
			return nil, err
		}
		if err := post.UpdateSlug(uniqueSlug); err != nil {
			return nil, ErrInvalidPostData.WithDetails(err.Error())
		}
	}

	// Save to repository
	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.Error(ctx, "failed to update post", "error", err, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to update post",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishPostUpdatedEvent(ctx, post)

	return post, nil
}

// PublishPost transitions a post to published status
func (s *PostsService) PublishPost(ctx context.Context, actorID uuid.UUID, id uuid.UUID) (*domain.Post, error) {
	// Check authorization - user must be able to publish this specific post
	canPublish, err := s.authorizer.Can(ctx, actorID, "posts", "publish", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canPublish {
		return nil, apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to publish this post",
			http.StatusForbidden,
		)
	}
	post, err := s.getPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := post.Publish(); err != nil {
		return nil, ErrInvalidStatusTransition.WithDetails(err.Error())
	}

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.Error(ctx, "failed to publish post", "error", err, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to publish post",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishPostPublishedEvent(ctx, post)

	return post, nil
}

// ArchivePost transitions a post to archived status
func (s *PostsService) ArchivePost(ctx context.Context, actorID uuid.UUID, id uuid.UUID) (*domain.Post, error) {
	// Check authorization - user must be able to archive this specific post
	canArchive, err := s.authorizer.Can(ctx, actorID, "posts", "archive", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canArchive {
		return nil, apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to archive this post",
			http.StatusForbidden,
		)
	}
	post, err := s.getPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := post.Archive(); err != nil {
		return nil, ErrInvalidStatusTransition.WithDetails(err.Error())
	}

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.Error(ctx, "failed to archive post", "error", err, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to archive post",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishPostArchivedEvent(ctx, post)

	return post, nil
}

// UnpublishPost transitions a post back to draft status
func (s *PostsService) UnpublishPost(ctx context.Context, actorID uuid.UUID, id uuid.UUID) (*domain.Post, error) {
	// Check authorization - user must be able to unpublish this specific post
	canUnpublish, err := s.authorizer.Can(ctx, actorID, "posts", "unpublish", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"authorization check failed",
			http.StatusInternalServerError,
		)
	}
	if !canUnpublish {
		return nil, apperror.New(
			apperror.CodeForbidden,
			apperror.BusinessCodePermissionDenied,
			"not authorized to unpublish this post",
			http.StatusForbidden,
		)
	}
	post, err := s.getPostByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := post.Unpublish(); err != nil {
		return nil, ErrInvalidStatusTransition.WithDetails(err.Error())
	}

	if err := s.repo.Update(ctx, post); err != nil {
		s.logger.Error(ctx, "failed to unpublish post", "error", err, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to unpublish post",
			http.StatusInternalServerError,
		)
	}

	// Publish event
	s.publishPostUpdatedEvent(ctx, post)

	return post, nil
}

// DeletePost removes a post from the system
func (s *PostsService) DeletePost(ctx context.Context, actorID uuid.UUID, id uuid.UUID) error {
	// Check authorization - user must be able to delete this specific post
	canDelete, err := s.authorizer.Can(ctx, actorID, "posts", "delete", &id)
	if err != nil {
		s.logger.Error(ctx, "failed to check authorization", "error", err, "actorID", actorID, "postID", id)
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
			"not authorized to delete this post",
			http.StatusForbidden,
		)
	}
	// Check if post exists
	post, err := s.getPostByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error(ctx, "failed to delete post", "error", err, "postID", id)
		return apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to delete post",
			http.StatusInternalServerError,
		)
	}

	// Publish event so other modules can clean up
	s.publishPostDeletedEvent(ctx, post)

	return nil
}

// GetPost retrieves a post by ID
func (s *PostsService) GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	return s.getPostByID(ctx, id)
}

// GetPostBySlug retrieves a post by its slug
func (s *PostsService) GetPostBySlug(ctx context.Context, slug string) (*domain.Post, error) {
	post, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ports.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		s.logger.Error(ctx, "failed to find post by slug", "error", err, "slug", slug)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to retrieve post",
			http.StatusInternalServerError,
		)
	}
	return post, nil
}

// ListPosts retrieves a list of post summaries
func (s *PostsService) ListPosts(ctx context.Context, filter ports.ListFilter) ([]*ports.PostSummary, int, error) {
	summaries, err := s.repo.ListSummaries(ctx, filter)
	if err != nil {
		s.logger.Error(ctx, "failed to list posts", "error", err)
		return nil, 0, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to list posts",
			http.StatusInternalServerError,
		)
	}

	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		s.logger.Error(ctx, "failed to count posts", "error", err)
		return nil, 0, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to count posts",
			http.StatusInternalServerError,
		)
	}

	return summaries, count, nil
}

// Private helper methods

// getPostByID fetches a post and handles not-found errors consistently
func (s *PostsService) getPostByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	post, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ports.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		s.logger.Error(ctx, "failed to find post", "error", err, "postID", id)
		return nil, apperror.New(
			apperror.CodeInternalError,
			apperror.BusinessCodeGeneral,
			"failed to retrieve post",
			http.StatusInternalServerError,
		)
	}
	return post, nil
}

func (s *PostsService) ensureUniqueSlug(ctx context.Context, baseSlug string, excludeID *uuid.UUID) (string, error) {
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
			return "", ErrSlugAlreadyExists.WithDetails(
				fmt.Sprintf("unable to generate unique slug for: %s", baseSlug),
			)
		}
	}
}

// Event publishing methods

func (s *PostsService) publishPostCreatedEvent(ctx context.Context, post *domain.Post) {
	event := eventbus.Event{
		Topic: events.PostCreatedTopic,
		Payload: events.PostCreatedEvent{
			PostID:     post.ID,
			ActorID:    post.AuthorID,
			Title:      post.Title,
			Slug:       post.Slug,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *PostsService) publishPostUpdatedEvent(ctx context.Context, post *domain.Post) {
	event := eventbus.Event{
		Topic: events.PostUpdatedTopic,
		Payload: events.PostUpdatedEvent{
			PostID:     post.ID,
			ActorID:    post.AuthorID, // In a real system, this might come from context
			Title:      post.Title,
			Slug:       post.Slug,
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *PostsService) publishPostPublishedEvent(ctx context.Context, post *domain.Post) {
	event := eventbus.Event{
		Topic: events.PostPublishedTopic,
		Payload: events.PostPublishedEvent{
			PostID:      post.ID,
			ActorID:     post.AuthorID, // In a real system, this might come from context
			PublishedAt: *post.PublishedAt,
			OccurredAt:  time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *PostsService) publishPostArchivedEvent(ctx context.Context, post *domain.Post) {
	event := eventbus.Event{
		Topic: events.PostArchivedTopic,
		Payload: events.PostArchivedEvent{
			PostID:     post.ID,
			ActorID:    post.AuthorID, // In a real system, this might come from context
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}

func (s *PostsService) publishPostDeletedEvent(ctx context.Context, post *domain.Post) {
	event := eventbus.Event{
		Topic: events.PostDeletedTopic,
		Payload: events.PostDeletedEvent{
			PostID:     post.ID,
			ActorID:    post.AuthorID, // In a real system, this might come from context
			OccurredAt: time.Now(),
		},
	}
	s.eventBus.Publish(ctx, event)
}
