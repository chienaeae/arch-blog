package application

import (
	"context"
	"errors"

	"backend/internal/platform/logger"
	"backend/internal/platform/ownership"
	"backend/internal/posts/ports"
	"github.com/google/uuid"
)

// PostsOwnershipChecker checks ownership of posts
// It depends directly on the repository, not the service, for cleaner architecture
type PostsOwnershipChecker struct {
	repo   ports.PostRepository
	logger logger.Logger
}

// NewPostsOwnershipChecker creates a new posts ownership checker
func NewPostsOwnershipChecker(repo ports.PostRepository, logger logger.Logger) *PostsOwnershipChecker {
	return &PostsOwnershipChecker{
		repo:   repo,
		logger: logger,
	}
}

// CheckOwnership checks if a user owns a specific post
// Implements the ownership.Checker interface
func (p *PostsOwnershipChecker) CheckOwnership(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (bool, error) {
	authorID, err := p.repo.GetPostAuthor(ctx, resourceID)
	if err != nil {
		if errors.Is(err, ports.ErrPostNotFound) {
			// Post doesn't exist, so user doesn't own it
			return false, nil
		}
		p.logger.Error(ctx, "failed to get post author", "error", err, "postID", resourceID)
		return false, err
	}

	return authorID == userID, nil
}

// RegisterPostsOwnership registers the posts ownership checker with the registry
func RegisterPostsOwnership(registry ownership.Registry, repo ports.PostRepository, logger logger.Logger) {
	checker := NewPostsOwnershipChecker(repo, logger)
	registry.RegisterChecker("posts", checker)
}
