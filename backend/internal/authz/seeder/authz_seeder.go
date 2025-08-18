package seeder

import (
	"context"
	"database/sql"
	"fmt"

	"backend/internal/authz/permission"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthzSeeder handles seeding of authorization data
type AuthzSeeder struct{}

// NewAuthzSeeder creates a new authorization seeder
func NewAuthzSeeder() *AuthzSeeder {
	return &AuthzSeeder{}
}

// Name returns the name of this seeder
func (s *AuthzSeeder) Name() string {
	return "AuthzSeeder"
}

// Seed runs the authorization seeding logic
func (s *AuthzSeeder) Seed(ctx context.Context, db *pgxpool.Pool) error {
	// Start a transaction for atomicity
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Seed permissions
	if err := s.seedPermissions(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	// Seed roles
	if err := s.seedRoles(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	// Seed role-permission mappings
	if err := s.seedRolePermissions(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed role permissions: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// seedPermissions inserts all permissions from the constants registry
func (s *AuthzSeeder) seedPermissions(ctx context.Context, tx pgx.Tx) error {
	// Get all permissions from the registry
	allPerms := permission.All()

	// Use batch insert for better performance
	batch := &pgx.Batch{}
	for _, perm := range allPerms {
		query := `
			INSERT INTO permissions (resource, action, scope, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (resource, action, scope) 
			DO UPDATE SET 
				description = EXCLUDED.description,
				updated_at = NOW()
		`
		// Handle NULL scope for permissions without scope
		var scope *string
		if perm.Scope != "" {
			scope = &perm.Scope
		}
		batch.Queue(query, perm.Resource, perm.Action, scope, perm.Description)
	}

	// Execute the batch
	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	// Check for errors
	for range allPerms {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert permission: %w", err)
		}
	}

	return br.Close()
}

// seedRoles inserts default roles from data.go
func (s *AuthzSeeder) seedRoles(ctx context.Context, tx pgx.Tx) error {
	// Use batch insert for better performance
	batch := &pgx.Batch{}

	for _, role := range DefaultRoles {
		query := `
			INSERT INTO roles (name, description, is_template, is_system)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) 
			DO UPDATE SET 
				description = EXCLUDED.description,
				updated_at = NOW()
		`
		batch.Queue(query, role.Name, role.Description, role.IsTemplate, role.IsSystem)
	}

	// Execute the batch
	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	// Check for errors
	for range DefaultRoles {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert role: %w", err)
		}
	}

	return br.Close()
}

// seedRolePermissions assigns permissions to roles with optimized batch processing
func (s *AuthzSeeder) seedRolePermissions(ctx context.Context, tx pgx.Tx) error {
	// Step 1: Fetch all role IDs into memory
	roleNameToID, err := s.fetchRoleMapping(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to fetch role mappings: %w", err)
	}

	// Step 2: Fetch all permission IDs into memory
	permissionIDToUUID, err := s.fetchPermissionMapping(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to fetch permission mappings: %w", err)
	}

	// Step 3: Prepare batch insert
	batch := &pgx.Batch{}

	// Handle super_admin specially - gets ALL permissions
	if superAdminID, exists := roleNameToID["super_admin"]; exists {
		for _, permUUID := range permissionIDToUUID {
			query := `
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES ($1, $2)
				ON CONFLICT (role_id, permission_id) DO NOTHING
			`
			batch.Queue(query, superAdminID, permUUID)
		}
	}

	// Handle other roles from DefaultRolePermissions
	for roleName, permissions := range DefaultRolePermissions {
		roleID, exists := roleNameToID[roleName]
		if !exists {
			return fmt.Errorf("role %s not found in database", roleName)
		}

		for _, permID := range permissions {
			permUUID, exists := permissionIDToUUID[permID]
			if !exists {
				return fmt.Errorf("permission %s not found in database", permID)
			}

			query := `
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES ($1, $2)
				ON CONFLICT (role_id, permission_id) DO NOTHING
			`
			batch.Queue(query, roleID, permUUID)
		}
	}

	// Step 4: Execute the batch
	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		defer func() { _ = br.Close() }()

		// Process all results
		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("failed to insert role permission: %w", err)
			}
		}

		return br.Close()
	}

	return nil
}

// fetchRoleMapping fetches all roles and returns a map of name -> UUID
func (s *AuthzSeeder) fetchRoleMapping(ctx context.Context, tx pgx.Tx) (map[string]uuid.UUID, error) {
	query := `SELECT id, name FROM roles`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roleMap := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		roleMap[name] = id
	}

	return roleMap, rows.Err()
}

// fetchPermissionMapping fetches all permissions and returns a map of permission ID string -> UUID
func (s *AuthzSeeder) fetchPermissionMapping(ctx context.Context, tx pgx.Tx) (map[string]uuid.UUID, error) {
	query := `
		SELECT id, resource, action, scope 
		FROM permissions
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permMap := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var resource, action string
		var scope sql.NullString
		if err := rows.Scan(&id, &resource, &action, &scope); err != nil {
			return nil, err
		}

		// Reconstruct the permission ID string
		permissionID := resource + ":" + action
		if scope.Valid && scope.String != "" {
			permissionID = permissionID + ":" + scope.String
		}

		permMap[permissionID] = id
	}

	return permMap, rows.Err()
}
