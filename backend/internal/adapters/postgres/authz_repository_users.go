package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
)

// ===== USER AUTHORIZATION OPERATIONS =====

// GetUserAuthz retrieves full authorization data for a user (for commands)
func (r *AuthzRepository) GetUserAuthz(ctx context.Context, userID uuid.UUID) (*domain.UserAuthz, error) {
	// Create the user authz object
	userAuthz := domain.NewUserAuthz(userID)

	// Get user's roles with their permissions using a single query
	roleQuery := `
		SELECT 
			r.id, r.name, r.description, r.is_template, r.is_system,
			r.created_at, r.updated_at,
			p.id, p.resource, p.action, p.scope, p.description,
			p.created_at, p.updated_at
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1
		ORDER BY r.name, p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, roleQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	// Map to track roles by ID
	roleMap := make(map[uuid.UUID]*domain.Role)

	for rows.Next() {
		// Role fields
		var roleID uuid.UUID
		var roleName, roleDesc string
		var isTemplate, isSystem bool
		var roleCreatedAt, roleUpdatedAt pgtype.Timestamptz

		// Permission fields (nullable since LEFT JOIN)
		var permID pgtype.UUID
		var permResource, permAction, permScope, permDesc pgtype.Text
		var permCreatedAt, permUpdatedAt pgtype.Timestamptz

		err := rows.Scan(
			&roleID, &roleName, &roleDesc, &isTemplate, &isSystem,
			&roleCreatedAt, &roleUpdatedAt,
			&permID, &permResource, &permAction, &permScope, &permDesc,
			&permCreatedAt, &permUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		// Get or create role
		role, exists := roleMap[roleID]
		if !exists {
			role = &domain.Role{
				ID:          roleID,
				Name:        roleName,
				Description: roleDesc,
				IsTemplate:  isTemplate,
				IsSystem:    isSystem,
				CreatedAt:   roleCreatedAt.Time,
				UpdatedAt:   roleUpdatedAt.Time,
				Permissions: make([]*domain.Permission, 0),
			}
			roleMap[roleID] = role
		}

		// Add permission if it exists (not NULL from LEFT JOIN)
		if permID.Valid {
			perm := &domain.Permission{
				ID:          uuid.UUID(permID.Bytes),
				Resource:    permResource.String,
				Action:      permAction.String,
				Description: permDesc.String,
				CreatedAt:   permCreatedAt.Time,
				UpdatedAt:   permUpdatedAt.Time,
			}
			if permScope.Valid {
				perm.Scope = permScope.String
			}
			role.Permissions = append(role.Permissions, perm)
		}
	}

	// Add roles to user authz
	for _, role := range roleMap {
		userAuthz.Roles = append(userAuthz.Roles, role)
	}

	// Get user's custom permissions
	customPermQuery := `
		SELECT 
			p.id, p.resource, p.action, p.scope, p.description,
			p.created_at, p.updated_at
		FROM user_permissions up
		JOIN permissions p ON up.permission_id = p.id
		WHERE up.user_id = $1
		ORDER BY p.resource, p.action, p.scope
	`

	permRows, err := r.db.Query(ctx, customPermQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user custom permissions: %w", err)
	}
	defer permRows.Close()

	for permRows.Next() {
		var perm domain.Permission
		var scope pgtype.Text

		err := permRows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&scope,
			&perm.Description,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan custom permission: %w", err)
		}

		if scope.Valid {
			perm.Scope = scope.String
		}

		userAuthz.CustomPermissions = append(userAuthz.CustomPermissions, &perm)
	}

	return userAuthz, nil
}

// AssignRoleToUser assigns a role to a user
func (r *AuthzRepository) AssignRoleToUser(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, grantedBy uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, granted_by, granted_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, role_id) 
		DO UPDATE SET 
			granted_by = EXCLUDED.granted_by,
			granted_at = EXCLUDED.granted_at
	`

	_, err := r.db.Exec(ctx, query, userID, roleID, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (r *AuthzRepository) RemoveRoleFromUser(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = $2
	`

	result, err := r.db.Exec(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user does not have this role")
	}

	return nil
}

// GrantPermissionToUser grants a custom permission to a user
func (r *AuthzRepository) GrantPermissionToUser(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID, grantedBy uuid.UUID) error {
	query := `
		INSERT INTO user_permissions (user_id, permission_id, granted_by, granted_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, permission_id) 
		DO UPDATE SET 
			granted_by = EXCLUDED.granted_by,
			granted_at = EXCLUDED.granted_at
	`

	_, err := r.db.Exec(ctx, query, userID, permissionID, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to grant permission to user: %w", err)
	}

	return nil
}

// RevokePermissionFromUser revokes a custom permission from a user
func (r *AuthzRepository) RevokePermissionFromUser(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID) error {
	query := `
		DELETE FROM user_permissions
		WHERE user_id = $1 AND permission_id = $2
	`

	result, err := r.db.Exec(ctx, query, userID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to revoke permission from user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user does not have this permission")
	}

	return nil
}

// ReplaceUserRoles replaces all user roles atomically
func (r *AuthzRepository) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, grantedBy uuid.UUID) error {
	// Start a transaction for atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete existing roles
	deleteQuery := `DELETE FROM user_roles WHERE user_id = $1`
	if _, err := tx.Exec(ctx, deleteQuery, userID); err != nil {
		return fmt.Errorf("failed to delete existing roles: %w", err)
	}

	// Insert new roles using batch
	if len(roleIDs) > 0 {
		batch := &pgx.Batch{}
		for _, roleID := range roleIDs {
			batch.Queue(
				"INSERT INTO user_roles (user_id, role_id, granted_by, granted_at) VALUES ($1, $2, $3, NOW())",
				userID, roleID, grantedBy,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(roleIDs); i++ {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to assign role: %w", err)
			}
		}
		if err := br.Close(); err != nil {
			return fmt.Errorf("failed to close batch: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ClearUserPermissions removes all custom permissions from a user
func (r *AuthzRepository) ClearUserPermissions(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_permissions WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to clear user permissions: %w", err)
	}

	return nil
}
