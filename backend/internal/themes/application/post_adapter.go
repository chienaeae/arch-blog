package application

import (
	"context"

	"github.com/google/uuid"
	postsApp "github.com/philly/arch-blog/backend/internal/posts/application"
	"github.com/philly/arch-blog/backend/internal/themes/domain"
)

// PostAdapter implements the PostProvider interface
// It adapts the posts service to provide PostInfo to the themes context
type PostAdapter struct {
	postsService *postsApp.PostsService
}

// NewPostAdapter creates a new post adapter
func NewPostAdapter(postsService *postsApp.PostsService) *PostAdapter {
	return &PostAdapter{
		postsService: postsService,
	}
}

// GetPost retrieves a post and returns it as PostInfo
func (a *PostAdapter) GetPost(ctx context.Context, id uuid.UUID) (domain.PostInfo, error) {
	post, err := a.postsService.GetPost(ctx, id)
	if err != nil {
		// Pass through the original error with all its rich information
		// The BaseHandler will handle AppErrors appropriately
		return nil, err
	}

	// The Post domain object directly implements PostInfo interface
	return post, nil
}
