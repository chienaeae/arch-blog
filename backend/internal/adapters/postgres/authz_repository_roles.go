package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
)

// ===== ROLE OPERATIONS =====

// GetRoleByID retrieves a role by its UUID (includes permissions)
func (r *AuthzRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	// Use a single query with LEFT JOIN to get role and permissions
	query := `
		SELECT 
			r.id, r.name, r.description, r.is_template, r.is_system, 
			r.created_at, r.updated_at,
			p.id, p.resource, p.action, p.scope, p.description, 
			p.created_at, p.updated_at
		FROM roles r
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN permissions p ON rp.permission_id = p.id
		WHERE r.id = $1
		ORDER BY p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	defer rows.Close()

	var role *domain.Role
	permissions := make([]*domain.Permission, 0)

	for rows.Next() {
		// Role fields
		var roleID uuid.UUID
		var roleName, roleDesc string
		var isTemplate, isSystem bool
		var roleCreatedAt, roleUpdatedAt pgtype.Timestamptz

		// Permission fields (nullable since LEFT JOIN)
		var permID pgtype.UUID
		var permResource, permAction, permDesc pgtype.Text
		var permScope pgtype.Text
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

		// Create role on first row
		if role == nil {
			role = &domain.Role{
				ID:          roleID,
				Name:        roleName,
				Description: roleDesc,
				IsTemplate:  isTemplate,
				IsSystem:    isSystem,
				CreatedAt:   roleCreatedAt.Time,
				UpdatedAt:   roleUpdatedAt.Time,
			}
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
			permissions = append(permissions, perm)
		}
	}

	if role == nil {
		return nil, fmt.Errorf("role not found")
	}

	role.Permissions = permissions
	return role, rows.Err()
}

// GetRoleByName retrieves a role by its name (includes permissions)
func (r *AuthzRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	// Use a single query with LEFT JOIN to get role and permissions
	query := `
		SELECT 
			r.id, r.name, r.description, r.is_template, r.is_system, 
			r.created_at, r.updated_at,
			p.id, p.resource, p.action, p.scope, p.description, 
			p.created_at, p.updated_at
		FROM roles r
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN permissions p ON rp.permission_id = p.id
		WHERE r.name = $1
		ORDER BY p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	defer rows.Close()

	var role *domain.Role
	permissions := make([]*domain.Permission, 0)

	for rows.Next() {
		// Role fields
		var roleID uuid.UUID
		var roleName, roleDesc string
		var isTemplate, isSystem bool
		var roleCreatedAt, roleUpdatedAt pgtype.Timestamptz

		// Permission fields (nullable since LEFT JOIN)
		var permID pgtype.UUID
		var permResource, permAction, permDesc pgtype.Text
		var permScope pgtype.Text
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

		// Create role on first row
		if role == nil {
			role = &domain.Role{
				ID:          roleID,
				Name:        roleName,
				Description: roleDesc,
				IsTemplate:  isTemplate,
				IsSystem:    isSystem,
				CreatedAt:   roleCreatedAt.Time,
				UpdatedAt:   roleUpdatedAt.Time,
			}
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
			permissions = append(permissions, perm)
		}
	}

	if role == nil {
		return nil, fmt.Errorf("role not found")
	}

	role.Permissions = permissions
	return role, rows.Err()
}

// GetAllRoles retrieves all roles in the system (optimized with single query)
func (r *AuthzRepository) GetAllRoles(ctx context.Context) ([]*domain.Role, error) {
	// Single query to get all roles and their permissions
	query := `
		SELECT 
			r.id, r.name, r.description, r.is_template, r.is_system, 
			r.created_at, r.updated_at,
			p.id, p.resource, p.action, p.scope, p.description, 
			p.created_at, p.updated_at
		FROM roles r
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN permissions p ON rp.permission_id = p.id
		ORDER BY r.name, p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	defer rows.Close()

	// Map to track roles by ID
	roleMap := make(map[uuid.UUID]*domain.Role)
	var roleOrder []uuid.UUID

	for rows.Next() {
		// Role fields
		var roleID uuid.UUID
		var roleName, roleDesc string
		var isTemplate, isSystem bool
		var roleCreatedAt, roleUpdatedAt pgtype.Timestamptz

		// Permission fields (nullable since LEFT JOIN)
		var permID pgtype.UUID
		var permResource, permAction, permDesc pgtype.Text
		var permScope pgtype.Text
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
			roleOrder = append(roleOrder, roleID)
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

	// Convert map to slice maintaining order
	roles := make([]*domain.Role, 0, len(roleOrder))
	for _, id := range roleOrder {
		roles = append(roles, roleMap[id])
	}

	return roles, rows.Err()
}

// GetRoleTemplates retrieves only template roles (optimized with single query)
func (r *AuthzRepository) GetRoleTemplates(ctx context.Context) ([]*domain.Role, error) {
	// Single query to get all template roles and their permissions
	query := `
		SELECT 
			r.id, r.name, r.description, r.is_template, r.is_system, 
			r.created_at, r.updated_at,
			p.id, p.resource, p.action, p.scope, p.description, 
			p.created_at, p.updated_at
		FROM roles r
		LEFT JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN permissions p ON rp.permission_id = p.id
		WHERE r.is_template = true
		ORDER BY r.name, p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get role templates: %w", err)
	}
	defer rows.Close()

	// Map to track roles by ID
	roleMap := make(map[uuid.UUID]*domain.Role)
	var roleOrder []uuid.UUID

	for rows.Next() {
		// Role fields
		var roleID uuid.UUID
		var roleName, roleDesc string
		var isTemplate, isSystem bool
		var roleCreatedAt, roleUpdatedAt pgtype.Timestamptz

		// Permission fields (nullable since LEFT JOIN)
		var permID pgtype.UUID
		var permResource, permAction, permDesc pgtype.Text
		var permScope pgtype.Text
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
			roleOrder = append(roleOrder, roleID)
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

	// Convert map to slice maintaining order
	roles := make([]*domain.Role, 0, len(roleOrder))
	for _, id := range roleOrder {
		roles = append(roles, roleMap[id])
	}

	return roles, rows.Err()
}

// CreateRole creates a new role (fully transactional)
func (r *AuthzRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	// Start a transaction for atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert the role
	query := `
		INSERT INTO roles (id, name, description, is_template, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.IsTemplate,
		role.IsSystem,
		role.CreatedAt,
		role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// If the role has permissions, add them using batch insert
	if len(role.Permissions) > 0 {
		batch := &pgx.Batch{}
		for _, perm := range role.Permissions {
			batch.Queue(
				"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)",
				role.ID, perm.ID,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(role.Permissions); i++ {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to assign permission to role: %w", err)
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

// UpdateRole updates an existing role (fully transactional, including permissions)
func (r *AuthzRepository) UpdateRole(ctx context.Context, role *domain.Role) error {
	// Start a transaction for atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update the role
	query := `
		UPDATE roles
		SET name = $2, description = $3, is_template = $4, is_system = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := tx.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.IsTemplate,
		role.IsSystem,
		role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}

	// Update permissions: delete existing and insert new ones
	deleteQuery := `DELETE FROM role_permissions WHERE role_id = $1`
	if _, err := tx.Exec(ctx, deleteQuery, role.ID); err != nil {
		return fmt.Errorf("failed to delete existing permissions: %w", err)
	}

	// Insert new permissions using batch
	if len(role.Permissions) > 0 {
		batch := &pgx.Batch{}
		for _, perm := range role.Permissions {
			batch.Queue(
				"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)",
				role.ID, perm.ID,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(role.Permissions); i++ {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to assign permission to role: %w", err)
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

// DeleteRole deletes a role by ID
func (r *AuthzRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM roles WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

// AssignPermissionsToRole assigns permissions to a role (replaces existing)
func (r *AuthzRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Start a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete existing permissions
	deleteQuery := `DELETE FROM role_permissions WHERE role_id = $1`
	if _, err := tx.Exec(ctx, deleteQuery, roleID); err != nil {
		return fmt.Errorf("failed to delete existing permissions: %w", err)
	}

	// Insert new permissions using batch
	if len(permissionIDs) > 0 {
		batch := &pgx.Batch{}
		for _, permID := range permissionIDs {
			batch.Queue(
				"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)",
				roleID, permID,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(permissionIDs); i++ {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to assign permission: %w", err)
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

// AddPermissionToRole adds a single permission to a role
func (r *AuthzRepository) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to add permission to role: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a single permission from a role
func (r *AuthzRepository) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	query := `
		DELETE FROM role_permissions
		WHERE role_id = $1 AND permission_id = $2
	`

	result, err := r.db.Exec(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("permission not found in role")
	}

	return nil
}