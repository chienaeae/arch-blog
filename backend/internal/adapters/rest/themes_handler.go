package rest

import (
	"encoding/json"
	"net/http"

	"backend/internal/adapters/api"
	"backend/internal/themes/application"
	"backend/internal/themes/domain"
	"backend/internal/themes/ports"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ThemesHandler handles HTTP requests for themes
type ThemesHandler struct {
	*BaseHandler
	service *application.ThemesService
}

// NewThemesHandler creates a new themes handler
func NewThemesHandler(base *BaseHandler, service *application.ThemesService) *ThemesHandler {
	return &ThemesHandler{
		BaseHandler: base,
		service:     service,
	}
}

// CreateTheme creates a new theme
// NOTE: Authorization is handled by middleware before this method is called
// Middleware ensures: 1) User is authenticated 2) User has themes:create permission
func (h *ThemesHandler) CreateTheme(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Parse request body
	var req api.CreateThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create the theme through the service
	params := application.CreateThemeParams{
		Name:        req.Name,
		Description: req.Description,
	}

	theme, err := h.service.CreateTheme(r.Context(), userID, params)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainThemeToAPI(theme)
	h.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// GetTheme retrieves a single theme by ID
// NOTE: Public endpoint - no authorization required
func (h *ThemesHandler) GetTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Get the theme
	theme, err := h.service.GetTheme(r.Context(), themeID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainThemeToAPI(theme)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetThemeBySlug retrieves a theme by its slug
// NOTE: Public endpoint - no authorization required
func (h *ThemesHandler) GetThemeBySlug(w http.ResponseWriter, r *http.Request, slug string) {
	// Get the theme
	theme, err := h.service.GetThemeBySlug(r.Context(), slug)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainThemeToAPI(theme)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// UpdateTheme updates an existing theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) UpdateTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Parse request body
	var req api.UpdateThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the theme through the service
	params := application.UpdateThemeParams{
		Name:        req.Name,
		Description: req.Description,
	}

	theme, err := h.service.UpdateTheme(r.Context(), userID, themeID, params)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response
	response := domainThemeToAPI(theme)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// DeleteTheme deletes a theme
// NOTE: Authorization middleware checks themes:delete:own permission before this is called
func (h *ThemesHandler) DeleteTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Delete the theme
	err := h.service.DeleteTheme(r.Context(), userID, themeID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// ActivateTheme activates a theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) ActivateTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Activate the theme
	err := h.service.ActivateTheme(r.Context(), userID, themeID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// DeactivateTheme deactivates a theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) DeactivateTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Deactivate the theme
	err := h.service.DeactivateTheme(r.Context(), userID, themeID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// ListThemes returns a paginated list of themes
// NOTE: Public endpoint - returns only active themes for anonymous users
func (h *ThemesHandler) ListThemes(w http.ResponseWriter, r *http.Request, params api.ListThemesParams) {
	// Build filter from query parameters
	filter := buildThemeListFilter(params)

	// Get themes and count
	themes, total, err := h.service.ListThemes(r.Context(), filter)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Reuse the common response building logic
	response := buildPaginatedThemesResponse(themes, total, filter)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetUserThemes returns themes created by a specific user
// NOTE: Public endpoint - shows only active themes unless requesting own themes
func (h *ThemesHandler) GetUserThemes(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Convert openapi UUID to google UUID
	userID := uuid.UUID(id)

	// Build filter with default pagination
	// Note: In a future API version, we could accept query params here
	filter := ports.ListFilter{
		CuratorID: &userID,
		Limit:     20,
		Offset:    0,
	}

	// Get themes and count
	themes, total, err := h.service.ListThemes(r.Context(), filter)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Reuse the common response building logic
	response := buildPaginatedThemesResponse(themes, total, filter)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetThemeWithArticles gets a theme with all its articles
// NOTE: Public endpoint - no authorization required
func (h *ThemesHandler) GetThemeWithArticles(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Get the theme with articles
	theme, err := h.service.GetTheme(r.Context(), themeID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert to API response with articles
	response := domainThemeWithArticlesToAPI(theme)
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// AddArticleToTheme adds an article to a theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) AddArticleToTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Parse request body
	var req api.AddArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert post ID
	postID := uuid.UUID(req.PostId)

	// Add article to theme
	err := h.service.AddArticleToTheme(r.Context(), userID, themeID, postID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// RemoveArticleFromTheme removes an article from a theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) RemoveArticleFromTheme(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, postId openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUIDs to google UUIDs
	themeUUID := uuid.UUID(id)
	postUUID := uuid.UUID(postId)

	// Remove article from theme
	err := h.service.RemoveArticleFromTheme(r.Context(), userID, themeUUID, postUUID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// ReorderThemeArticles reorders articles within a theme
// NOTE: Authorization middleware checks themes:update:own permission before this is called
func (h *ThemesHandler) ReorderThemeArticles(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	// Get authenticated user ID - middleware guarantees this exists
	userID := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	themeID := uuid.UUID(id)

	// Parse request body
	var req api.ReorderArticlesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build new ordering from request - using the post IDs in the order provided
	postIDs := make([]uuid.UUID, len(req.PostIds))
	for i, postID := range req.PostIds {
		postIDs[i] = uuid.UUID(postID)
	}

	// Reorder articles
	err := h.service.ReorderThemeArticles(r.Context(), userID, themeID, postIDs)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func buildPaginatedThemesResponse(themes []*ports.ThemeSummary, total int, filter ports.ListFilter) api.PaginatedThemes {
	// Convert to API response
	apiThemes := make([]api.ThemeSummary, len(themes))
	for i, theme := range themes {
		apiThemes[i] = themeSummaryToAPI(theme)
	}

	// Calculate pagination metadata
	itemsPerPage := filter.Limit
	if itemsPerPage == 0 {
		itemsPerPage = 20
	}
	currentPage := (filter.Offset / itemsPerPage) + 1
	totalPages := (total + itemsPerPage - 1) / itemsPerPage

	return api.PaginatedThemes{
		Data: apiThemes,
		Meta: api.PaginationMeta{
			TotalItems:   total,
			ItemsPerPage: itemsPerPage,
			CurrentPage:  currentPage,
			TotalPages:   totalPages,
		},
	}
}

func buildThemeListFilter(params api.ListThemesParams) ports.ListFilter {
	filter := ports.ListFilter{
		Limit:  20,
		Offset: 0,
	}

	// Pagination - convert page-based to offset-based
	if params.Limit != nil {
		filter.Limit = *params.Limit
	}
	if params.Page != nil && *params.Page > 0 {
		filter.Offset = (*params.Page - 1) * filter.Limit
	}

	// IsActive filter
	if params.IsActive != nil {
		filter.IsActive = params.IsActive
	}

	// Curator filter
	if params.CuratorId != nil {
		curatorID := uuid.UUID(*params.CuratorId)
		filter.CuratorID = &curatorID
	}

	// Note: Sorting is not implemented in the repository yet
	// This would need to be added to the ListFilter and repository implementation

	return filter
}

func themeSummaryToAPI(summary *ports.ThemeSummary) api.ThemeSummary {
	apiSummary := api.ThemeSummary{
		Id:           openapi_types.UUID(summary.ID),
		Name:         summary.Name,
		Description:  summary.Description,
		Slug:         summary.Slug,
		IsActive:     summary.IsActive,
		CuratorId:    openapi_types.UUID(summary.CuratorID),
		CreatedAt:    summary.CreatedAt,
		ArticleCount: summary.ArticleCount,
	}

	return apiSummary
}

func domainThemeToAPI(theme *domain.Theme) api.Theme {
	apiTheme := api.Theme{
		Id:           openapi_types.UUID(theme.ID),
		Name:         theme.Name,
		Description:  theme.Description,
		Slug:         theme.Slug,
		IsActive:     theme.IsActive,
		CuratorId:    openapi_types.UUID(theme.CuratorID),
		CreatedAt:    theme.CreatedAt,
		UpdatedAt:    theme.UpdatedAt,
		ArticleCount: len(theme.Articles),
	}

	return apiTheme
}

func domainThemeWithArticlesToAPI(theme *domain.Theme) api.ThemeWithArticles {
	apiTheme := api.ThemeWithArticles{
		Id:           openapi_types.UUID(theme.ID),
		Name:         theme.Name,
		Description:  theme.Description,
		Slug:         theme.Slug,
		IsActive:     theme.IsActive,
		CuratorId:    openapi_types.UUID(theme.CuratorID),
		CreatedAt:    theme.CreatedAt,
		UpdatedAt:    theme.UpdatedAt,
		ArticleCount: len(theme.Articles),
		Articles:     make([]api.ThemeArticle, 0, len(theme.Articles)),
	}

	// Convert articles
	for _, article := range theme.Articles {
		apiTheme.Articles = append(apiTheme.Articles, api.ThemeArticle{
			PostId:   openapi_types.UUID(article.PostID),
			Position: article.Position,
			AddedAt:  article.AddedAt,
			AddedBy:  openapi_types.UUID(article.AddedBy),
		})
	}

	return apiTheme
}
