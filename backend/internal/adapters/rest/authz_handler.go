package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/philly/arch-blog/backend/internal/adapters/api"
	"github.com/philly/arch-blog/backend/internal/authz/application"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
)

// AuthzHandler handles authorization management endpoints
type AuthzHandler struct {
	*BaseHandler
	service *application.AuthzService
}

// NewAuthzHandler creates a new authorization handler
func NewAuthzHandler(base *BaseHandler, service *application.AuthzService) *AuthzHandler {
	return &AuthzHandler{
		BaseHandler: base,
		service:     service,
	}
}


// ListPermissions returns all available permissions in the system
func (h *AuthzHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	permissions, err := h.service.GetAllPermissions(ctx)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Map domain permissions to API permissions
	apiPermissions := make([]api.Permission, len(permissions))
	for i, perm := range permissions {
		apiPermissions[i] = h.mapDomainPermissionToAPI(perm)
	}

	h.WriteJSONResponse(w, r, apiPermissions, http.StatusOK)
}

// ListRoles returns all roles in the system
func (h *AuthzHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	roles, err := h.service.GetAllRoles(ctx)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Map domain roles to API roles
	apiRoles := make([]api.Role, len(roles))
	for i, role := range roles {
		apiRoles[i] = h.mapDomainRoleToAPI(role)
	}

	h.WriteJSONResponse(w, r, apiRoles, http.StatusOK)
}

// CreateRole creates a new role
func (h *AuthzHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode and validate request
	var req api.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user ID for audit purposes (even though not used in service yet)
	_ = h.GetUserIDFromContext(r)

	// Basic validation
	if req.Name == "" || req.Description == "" {
		h.WriteJSONError(w, r, "validation_error", "Name and description are required", http.StatusBadRequest)
		return
	}

	// Determine if this should be a template
	isTemplate := false
	if req.IsTemplate != nil {
		isTemplate = *req.IsTemplate
	}

	// Create the role (permissions must be added separately after creation)
	// The service only supports creating roles without initial permissions
	role, err := h.service.CreateRole(ctx, req.Name, req.Description, isTemplate)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, h.mapDomainRoleToAPI(role), http.StatusCreated)
}

// GetRole returns a single role by ID - implements OpenAPI interface
func (h *AuthzHandler) GetRole(w http.ResponseWriter, r *http.Request, roleId openapi_types.UUID) {
	ctx := r.Context()

	// Convert openapi UUID to google UUID
	roleUUID := uuid.UUID(roleId)

	// Get the role
	role, err := h.service.GetRole(ctx, roleUUID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, h.mapDomainRoleToAPI(role), http.StatusOK)
}

// UpdateRole updates a role's name and description
func (h *AuthzHandler) UpdateRole(w http.ResponseWriter, r *http.Request, roleId openapi_types.UUID) {
	ctx := r.Context()

	// Convert openapi UUID to google UUID
	roleUUID := uuid.UUID(roleId)

	// Decode request
	var req api.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the role (service handles partial updates)
	role, err := h.service.UpdateRole(ctx, roleUUID, req.Name, req.Description)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, h.mapDomainRoleToAPI(role), http.StatusOK)
}

// DeleteRole deletes a role
func (h *AuthzHandler) DeleteRole(w http.ResponseWriter, r *http.Request, roleId openapi_types.UUID) {
	ctx := r.Context()

	// Get current user ID for audit purposes (even though not used in service yet)
	_ = h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	roleUUID := uuid.UUID(roleId)

	// Delete the role (service only needs roleID, not currentUserID)
	if err := h.service.DeleteRole(ctx, roleUUID); err != nil {
		h.HandleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateRolePermissions replaces all permissions for a role
func (h *AuthzHandler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request, roleId openapi_types.UUID) {
	ctx := r.Context()

	// Convert openapi UUID to google UUID
	roleUUID := uuid.UUID(roleId)

	// Decode request
	var req api.RolePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse permission IDs
	permissionIDs := make([]uuid.UUID, 0, len(req.Permissions))
	for _, permID := range req.Permissions {
		// Convert openapi UUID to google UUID directly
		permissionIDs = append(permissionIDs, uuid.UUID(permID))
	}

	// Update role permissions
	role, err := h.service.UpdateRolePermissions(ctx, roleUUID, permissionIDs)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, h.mapDomainRoleToAPI(role), http.StatusOK)
}

// GetUserRoles returns all roles assigned to a user
func (h *AuthzHandler) GetUserRoles(w http.ResponseWriter, r *http.Request, userId openapi_types.UUID) {
	ctx := r.Context()

	// Convert openapi UUID to google UUID
	userUUID := uuid.UUID(userId)

	// Get user roles with full details
	userRoles, err := h.service.GetUserRolesWithDetails(ctx, userUUID)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Map to API response
	apiUserRoles := make([]api.UserRole, len(userRoles))
	for i, ur := range userRoles {
		apiUserRoles[i] = api.UserRole{
			UserId:    openapi_types.UUID(ur.UserID),
			RoleId:    openapi_types.UUID(ur.RoleID),
			Role:      h.mapDomainRoleToAPI(ur.Role),
			GrantedAt: ur.GrantedAt,
		}
		if ur.GrantedBy != uuid.Nil {
			grantedBy := openapi_types.UUID(ur.GrantedBy)
			apiUserRoles[i].GrantedBy = &grantedBy
		}
	}

	h.WriteJSONResponse(w, r, apiUserRoles, http.StatusOK)
}

// AssignRoleToUser assigns a role to a user
func (h *AuthzHandler) AssignRoleToUser(w http.ResponseWriter, r *http.Request, userId openapi_types.UUID) {
	ctx := r.Context()

	// Get current user ID (who is granting the role)
	grantedBy := h.GetUserIDFromContext(r)

	// Convert openapi UUID to google UUID
	userUUID := uuid.UUID(userId)

	// Decode request
	var req api.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "validation_error", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert openapi UUID to google UUID 
	roleUUID := uuid.UUID(req.RoleId)

	// Assign the role
	err := h.service.AssignRoleToUser(ctx, userUUID, roleUUID, grantedBy)
	if err != nil {
		h.HandleError(w, r, err)
		return
	}

	// Return success with minimal response
	w.WriteHeader(http.StatusCreated)
}

// RevokeRoleFromUser removes a role from a user
func (h *AuthzHandler) RevokeRoleFromUser(w http.ResponseWriter, r *http.Request, userId openapi_types.UUID, roleId openapi_types.UUID) {
	ctx := r.Context()

	// Convert openapi UUIDs to google UUIDs
	userUUID := uuid.UUID(userId)
	roleUUID := uuid.UUID(roleId)

	// Revoke the role
	if err := h.service.RemoveRoleFromUser(ctx, userUUID, roleUUID); err != nil {
		h.HandleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Mapper functions to convert domain models to API models

func (h *AuthzHandler) mapDomainPermissionToAPI(perm *domain.Permission) api.Permission {
	apiPerm := api.Permission{
		Id:        openapi_types.UUID(perm.ID),
		Resource:  perm.Resource,
		Action:    perm.Action,
		CreatedAt: perm.CreatedAt,
		UpdatedAt: perm.UpdatedAt,
	}

	if perm.Scope != "" {
		apiPerm.Scope = &perm.Scope
	}
	if perm.Description != "" {
		apiPerm.Description = &perm.Description
	}

	return apiPerm
}

func (h *AuthzHandler) mapDomainRoleToAPI(role *domain.Role) api.Role {
	permissions := make([]api.Permission, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissions[i] = h.mapDomainPermissionToAPI(perm)
	}

	return api.Role{
		Id:          openapi_types.UUID(role.ID),
		Name:        role.Name,
		Description: role.Description,
		IsTemplate:  role.IsTemplate,
		IsSystem:    role.IsSystem,
		Permissions: permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

