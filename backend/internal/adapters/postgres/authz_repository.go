package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
)

// AuthzRepository implements the authorization repository using PostgreSQL
// It acts as a pure data mapper without any domain logic
type AuthzRepository struct {
	db *pgxpool.Pool
}

// NewAuthzRepository creates a new PostgreSQL authorization repository
func NewAuthzRepository(db *pgxpool.Pool) *AuthzRepository {
	return &AuthzRepository{
		db: db,
	}
}

// ===== OPTIMIZED QUERY OPERATIONS =====
// These are the most performance-critical methods

// HasPermission checks if a user has a specific permission (optimized query)
func (r *AuthzRepository) HasPermission(ctx context.Context, userID uuid.UUID, permissionID string) (bool, error) {
	// Parse the permission ID to get components
	resource, action, scope := domain.ParsePermissionID(permissionID)

	query := `
		SELECT EXISTS (
			-- Check permissions from roles
			SELECT 1 
			FROM user_roles ur
			JOIN role_permissions rp ON ur.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE ur.user_id = $1 
				AND p.resource = $2 
				AND p.action = $3
				AND (p.scope = $4 OR ($4 IS NULL AND p.scope IS NULL))
			
			UNION
			
			-- Check direct user permissions
			SELECT 1
			FROM user_permissions up
			JOIN permissions p ON up.permission_id = p.id
			WHERE up.user_id = $1 
				AND p.resource = $2 
				AND p.action = $3
				AND (p.scope = $4 OR ($4 IS NULL AND p.scope IS NULL))
		)
	`

	var scopeParam pgtype.Text
	if scope != "" {
		scopeParam = pgtype.Text{String: scope, Valid: true}
	}

	var hasPermission bool
	err := r.db.QueryRow(ctx, query, userID, resource, action, scopeParam).Scan(&hasPermission)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return hasPermission, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *AuthzRepository) HasAnyPermission(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error) {
	// Build the query with dynamic WHERE clauses for each permission
	queryBase := `
		SELECT EXISTS (
			SELECT 1 FROM (
				-- Check permissions from roles
				SELECT p.resource, p.action, p.scope
				FROM user_roles ur
				JOIN role_permissions rp ON ur.role_id = rp.role_id
				JOIN permissions p ON rp.permission_id = p.id
				WHERE ur.user_id = $1
				
				UNION
				
				-- Check direct user permissions
				SELECT p.resource, p.action, p.scope
				FROM user_permissions up
				JOIN permissions p ON up.permission_id = p.id
				WHERE up.user_id = $1
			) AS user_perms
			WHERE `

	// Build WHERE conditions for each permission
	conditions := make([]string, 0, len(permissionIDs))
	args := []interface{}{userID}
	argCount := 1

	for _, permID := range permissionIDs {
		resource, action, scope := domain.ParsePermissionID(permID)
		argCount++
		resourceArg := argCount
		argCount++
		actionArg := argCount

		if scope == "" {
			conditions = append(conditions, fmt.Sprintf(
				"(user_perms.resource = $%d AND user_perms.action = $%d AND user_perms.scope IS NULL)",
				resourceArg, actionArg))
			args = append(args, resource, action)
		} else {
			argCount++
			scopeArg := argCount
			conditions = append(conditions, fmt.Sprintf(
				"(user_perms.resource = $%d AND user_perms.action = $%d AND user_perms.scope = $%d)",
				resourceArg, actionArg, scopeArg))
			args = append(args, resource, action, scope)
		}
	}

	query := queryBase
	for i, cond := range conditions {
		if i > 0 {
			query += " OR "
		}
		query += cond
	}
	query += ")"

	var hasAny bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&hasAny)
	if err != nil {
		return false, fmt.Errorf("failed to check any permissions: %w", err)
	}

	return hasAny, nil
}

// HasAllPermissions checks if a user has all of the specified permissions
func (r *AuthzRepository) HasAllPermissions(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error) {
	// For each permission, check if user has it
	for _, permID := range permissionIDs {
		hasPermission, err := r.HasPermission(ctx, userID, permID)
		if err != nil {
			return false, err
		}
		if !hasPermission {
			return false, nil
		}
	}
	return true, nil
}

// HasRole checks if a user has a specific role
func (r *AuthzRepository) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`

	var hasRole bool
	err := r.db.QueryRow(ctx, query, userID, roleName).Scan(&hasRole)
	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}

	return hasRole, nil
}

// GetUserPermissionIDs gets all permission IDs for a user (optimized)
func (r *AuthzRepository) GetUserPermissionIDs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT p.resource, p.action, p.scope
		FROM (
			-- Get permissions from roles
			SELECT p.resource, p.action, p.scope
			FROM user_roles ur
			JOIN role_permissions rp ON ur.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE ur.user_id = $1
			
			UNION
			
			-- Get direct user permissions
			SELECT p.resource, p.action, p.scope
			FROM user_permissions up
			JOIN permissions p ON up.permission_id = p.id
			WHERE up.user_id = $1
		) AS p
		ORDER BY p.resource, p.action, p.scope
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var resource, action string
		var scope pgtype.Text
		if err := rows.Scan(&resource, &action, &scope); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		// Build the permission ID string
		permID := resource + ":" + action
		if scope.Valid && scope.String != "" {
			permID = permID + ":" + scope.String
		}
		permissions = append(permissions, permID)
	}

	return permissions, rows.Err()
}

// GetUserRoleNames gets all role names for a user (optimized)
func (r *AuthzRepository) GetUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, roleName)
	}

	return roles, rows.Err()
}

// ===== PERMISSION OPERATIONS =====

// GetPermissionByID retrieves a permission by its UUID
func (r *AuthzRepository) GetPermissionByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	query := `
		SELECT id, resource, action, scope, description, created_at, updated_at
		FROM permissions
		WHERE id = $1
	`

	var perm domain.Permission
	var scope pgtype.Text
	err := r.db.QueryRow(ctx, query, id).Scan(
		&perm.ID,
		&perm.Resource,
		&perm.Action,
		&scope,
		&perm.Description,
		&perm.CreatedAt,
		&perm.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("permission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	if scope.Valid {
		perm.Scope = scope.String
	}

	return &perm, nil
}

// GetPermissionByIDString retrieves a permission by its string identifier
func (r *AuthzRepository) GetPermissionByIDString(ctx context.Context, permissionID string) (*domain.Permission, error) {
	// Parse the permission ID to get components
	resource, action, scope := domain.ParsePermissionID(permissionID)

	query := `
		SELECT id, resource, action, scope, description, created_at, updated_at
		FROM permissions
		WHERE resource = $1 AND action = $2 AND (scope = $3 OR ($3 IS NULL AND scope IS NULL))
	`

	var scopeParam *string
	if scope != "" {
		scopeParam = &scope
	}

	var perm domain.Permission
	var dbScope pgtype.Text
	err := r.db.QueryRow(ctx, query, resource, action, scopeParam).Scan(
		&perm.ID,
		&perm.Resource,
		&perm.Action,
		&dbScope,
		&perm.Description,
		&perm.CreatedAt,
		&perm.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("permission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	if dbScope.Valid {
		perm.Scope = dbScope.String
	}

	return &perm, nil
}

// GetAllPermissions retrieves all permissions in the system
func (r *AuthzRepository) GetAllPermissions(ctx context.Context) ([]*domain.Permission, error) {
	query := `
		SELECT id, resource, action, scope, description, created_at, updated_at
		FROM permissions
		ORDER BY resource, action, scope
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*domain.Permission
	for rows.Next() {
		var perm domain.Permission
		var scope pgtype.Text
		if err := rows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&scope,
			&perm.Description,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		if scope.Valid {
			perm.Scope = scope.String
		}

		permissions = append(permissions, &perm)
	}

	return permissions, rows.Err()
}

// CreatePermission creates a new permission
func (r *AuthzRepository) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	query := `
		INSERT INTO permissions (id, resource, action, scope, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var scope *string
	if permission.Scope != "" {
		scope = &permission.Scope
	}

	_, err := r.db.Exec(ctx, query,
		permission.ID,
		permission.Resource,
		permission.Action,
		scope,
		permission.Description,
		permission.CreatedAt,
		permission.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

// UpdatePermission updates an existing permission
func (r *AuthzRepository) UpdatePermission(ctx context.Context, permission *domain.Permission) error {
	query := `
		UPDATE permissions
		SET resource = $2, action = $3, scope = $4, description = $5, updated_at = $6
		WHERE id = $1
	`

	var scope *string
	if permission.Scope != "" {
		scope = &permission.Scope
	}

	result, err := r.db.Exec(ctx, query,
		permission.ID,
		permission.Resource,
		permission.Action,
		scope,
		permission.Description,
		permission.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

// DeletePermission deletes a permission by ID
func (r *AuthzRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM permissions WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}
