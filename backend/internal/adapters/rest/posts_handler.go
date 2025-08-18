package rest

import (
	"encoding/json"
	"net/http"

	"backend/internal/adapters/api"
	"backend/internal/posts/application"
	"backend/internal/posts/domain"
	"backend/internal/posts/ports"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// PostsHandler handles HTTP requests for posts
type PostsHandler struct {
	*BaseHandler
	service *application.PostsService
}

// NewPostsHandler creates a new posts handler
func NewPostsHandler(base *BaseHandler, service *application.PostsService) *PostsHandler {
	return &PostsHandler{
		BaseHandler: base,
		service:     service,
	}
}

// CreatePost creates a new blog post
// NOTE: Authorization is handled by middleware before this method is called
// Middleware ensures: 1) User is authenticated 2) User has posts:create permission
func (h *PostsHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Parse request body
	var req api.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create the post through the service
	params := application.CreatePostParams{
		Title:   req.Title,
		Content: req.Content,
		Excerpt: req.Excerpt,
	}

	post, err := h.service.CreatePost(r.Context(), userID, params)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// GetPost retrieves a single post by ID
// NOTE: Public endpoint - no authorization required
func (h *PostsHandler) GetPost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Get the post
	post, err := h.service.GetPost(r.Context(), postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetPostBySlug retrieves a post by its slug
// NOTE: Public endpoint - no authorization required
func (h *PostsHandler) GetPostBySlug(w http.ResponseWriter, r *http.Request, slug string) {
	// Get the post
	post, err := h.service.GetPostBySlug(r.Context(), slug)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// UpdatePost updates an existing post
// NOTE: Authorization middleware checks posts:update:own permission before this is called
func (h *PostsHandler) UpdatePost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Parse request body
	var req api.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the post through the service
	params := application.UpdatePostParams{
		Title:   req.Title,
		Content: req.Content,
		Excerpt: req.Excerpt,
	}

	post, err := h.service.UpdatePost(r.Context(), userID, postID, params)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// PublishPost publishes a draft post
// NOTE: Authorization middleware checks posts:publish:own permission before this is called
func (h *PostsHandler) PublishPost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Publish the post
	post, err := h.service.PublishPost(r.Context(), userID, postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// UnpublishPost unpublishes a published post (back to draft)
// NOTE: Authorization middleware checks posts:publish:own permission before this is called
func (h *PostsHandler) UnpublishPost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Unpublish the post
	post, err := h.service.UnpublishPost(r.Context(), userID, postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// ArchivePost archives a post
// NOTE: Authorization middleware checks posts:archive:own permission before this is called
func (h *PostsHandler) ArchivePost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Archive the post
	post, err := h.service.ArchivePost(r.Context(), userID, postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success response
	response := domainPostToAPI(post)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// DeletePost deletes a post
// NOTE: Authorization middleware checks posts:delete:own permission before this is called
func (h *PostsHandler) DeletePost(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	postID := uuid.UUID(id)

	// Delete the post
	err := h.service.DeletePost(r.Context(), userID, postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// ListPosts returns a paginated list of posts
// NOTE: Public endpoint - returns only published posts for anonymous users
func (h *PostsHandler) ListPosts(w http.ResponseWriter, r *http.Request, params api.ListPostsParams) {
	// Build filter from query parameters
	filter := buildListFilter(params)

	// Get posts and count
	summaries, total, err := h.service.ListPosts(r.Context(), filter)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Reuse the common response building logic
	response := buildPaginatedPostsResponse(summaries, total, filter)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetUserPosts returns posts by a specific user
// NOTE: Public endpoint - shows only published posts unless requesting own posts
func (h *PostsHandler) GetUserPosts(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Convert openapi UUID to google UUID
	userID := uuid.UUID(id)

	// Build filter with default pagination
	// Note: In a future API version, we could accept query params here
	filter := ports.DefaultListFilter()
	filter.AuthorID = &userID

	// Get posts and count
	summaries, total, err := h.service.ListPosts(r.Context(), filter)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Reuse the common response building logic
	response := buildPaginatedPostsResponse(summaries, total, filter)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// Helper functions

func buildPaginatedPostsResponse(summaries []*ports.PostSummary, total int, filter ports.ListFilter) api.PaginatedPosts {
	// Convert to API response
	apiSummaries := make([]api.PostSummary, len(summaries))
	for i, summary := range summaries {
		apiSummaries[i] = domainSummaryToAPI(summary)
	}

	// Calculate pagination metadata
	itemsPerPage := filter.Limit
	if itemsPerPage == 0 {
		itemsPerPage = 20
	}
	currentPage := (filter.Offset / itemsPerPage) + 1
	totalPages := (total + itemsPerPage - 1) / itemsPerPage

	return api.PaginatedPosts{
		Data: apiSummaries,
		Meta: api.PaginationMeta{
			TotalItems:   total,
			ItemsPerPage: itemsPerPage,
			CurrentPage:  currentPage,
			TotalPages:   totalPages,
		},
	}
}

func buildListFilter(params api.ListPostsParams) ports.ListFilter {
	filter := ports.DefaultListFilter()

	// Pagination - convert page-based to offset-based
	if params.Limit != nil {
		filter.Limit = *params.Limit
	}
	if params.Page != nil && *params.Page > 0 {
		filter.Offset = (*params.Page - 1) * filter.Limit
	}

	// Status filter
	if params.Status != nil {
		status := domain.PostStatus(*params.Status)
		filter.Status = &status
	}

	// Author filter
	if params.AuthorId != nil {
		authorID := uuid.UUID(*params.AuthorId)
		filter.AuthorID = &authorID
	}

	// Note: The API doesn't have a search parameter yet, but the filter supports it
	// This could be added to the OpenAPI spec if needed

	// Sorting
	if params.SortBy != nil {
		switch *params.SortBy {
		case api.ListPostsParamsSortByCreatedAt:
			filter.OrderBy = ports.OrderByCreatedAt
		case api.ListPostsParamsSortByUpdatedAt:
			filter.OrderBy = ports.OrderByUpdatedAt
		case api.ListPostsParamsSortByPublishedAt:
			filter.OrderBy = ports.OrderByPublishedAt
		case api.ListPostsParamsSortByTitle:
			filter.OrderBy = ports.OrderByTitle
		}
	}

	if params.SortOrder != nil && *params.SortOrder == api.ListPostsParamsSortOrderAsc {
		filter.OrderDesc = false
	}

	return filter
}

func domainPostToAPI(post *domain.Post) api.Post {
	apiPost := api.Post{
		Id:        openapi_types.UUID(post.ID),
		Title:     post.Title,
		Content:   post.Content,
		Excerpt:   post.Excerpt,
		Slug:      post.Slug,
		Status:    api.PostStatus(post.Status),
		AuthorId:  openapi_types.UUID(post.AuthorID),
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}

	if post.PublishedAt != nil {
		apiPost.PublishedAt = post.PublishedAt
	}

	return apiPost
}

func domainSummaryToAPI(summary *ports.PostSummary) api.PostSummary {
	apiSummary := api.PostSummary{
		Id:        openapi_types.UUID(summary.ID),
		Title:     summary.Title,
		Excerpt:   summary.Excerpt,
		Slug:      summary.Slug,
		Status:    api.PostSummaryStatus(summary.Status),
		AuthorId:  openapi_types.UUID(summary.AuthorID),
		CreatedAt: summary.CreatedAt,
		ViewCount: 0, // View count not tracked yet
	}

	// Set published date - use created date as fallback if not published
	if summary.PublishedAt != nil {
		apiSummary.PublishedAt = *summary.PublishedAt
	} else {
		apiSummary.PublishedAt = summary.CreatedAt
	}

	return apiSummary
}
