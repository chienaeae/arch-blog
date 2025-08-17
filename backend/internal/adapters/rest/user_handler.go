package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/philly/arch-blog/backend/internal/adapters/api"
	"github.com/philly/arch-blog/backend/internal/adapters/rest/middleware"
	"github.com/philly/arch-blog/backend/internal/users/application"
	"github.com/philly/arch-blog/backend/internal/users/domain"
)

type UserHandler struct {
	*BaseHandler
	service *application.UserService
}

func NewUserHandler(base *BaseHandler, service *application.UserService) *UserHandler {
	return &UserHandler{
		BaseHandler: base,
		service:     service,
	}
}

// CreateUser implements the OpenAPI generated ServerInterface
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Extract JWT claims directly (not internal user ID) because this endpoint
	// creates the user profile for the first time. The user doesn't exist in our
	// database yet, so AuthAdapter cannot resolve to an internal ID.
	// This is the ONLY handler that should use JWT claims directly.
	supabaseID, ok := middleware.GetJWTUserID(r.Context())
	if !ok {
		h.WriteJSONError(w, r, "unauthorized", "User ID not found in context", http.StatusUnauthorized)
		return
	}

	email, ok := middleware.GetJWTUserEmail(r.Context())
	if !ok {
		h.WriteJSONError(w, r, "unauthorized", "Email not found in context", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req api.NewUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build service parameters
	params := application.CreateUserParams{
		SupabaseID:  supabaseID,
		Email:       email,
		Username:    req.Username,
		DisplayName: getStringValue(req.DisplayName),
		Bio:         getStringValue(req.Bio),
		AvatarURL:   getStringValue(req.AvatarUrl),
	}

	// Call service to create user (validation happens in the service)
	user, err := h.service.CreateUser(r.Context(), params)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert domain user to API response
	response := domainUserToAPI(user)

	// Return success response
	h.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// GetCurrentUser implements the OpenAPI generated ServerInterface
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Extract internal user ID from context (set by AuthAdapter middleware)
	// This is the canonical internal UUID, not the external Supabase ID
	userID := h.GetUserIDFromContext(r)

	// Get user from service using internal ID
	user, err := h.service.GetUserByID(r.Context(), userID.String())
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Convert domain user to API response
	response := domainUserToAPI(user)

	// Return success response
	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// Helper function to convert domain User to API User
func domainUserToAPI(user *domain.User) api.User {
	// Parse UUID string (User.ID is a string, not uuid.UUID)
	parsedUUID, _ := uuid.Parse(user.ID)

	return api.User{
		Id:          openapi_types.UUID(parsedUUID),
		Email:       openapi_types.Email(user.Email),
		Username:    user.Username,
		DisplayName: stringToPointer(user.DisplayName),
		Bio:         stringToPointer(user.Bio),
		AvatarUrl:   stringToPointer(user.AvatarURL),
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
