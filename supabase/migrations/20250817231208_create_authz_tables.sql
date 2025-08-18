-- V2__create_authz_tables.sql
-- Authorization system tables for RBAC with role templates and custom permissions

-- Permissions table: Defines all possible permissions in the system
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(50) NOT NULL,    -- e.g., "posts", "users", "comments"
    action VARCHAR(50) NOT NULL,      -- e.g., "create", "read", "update", "delete"
    scope VARCHAR(20),                -- e.g., "own", "any", "self", or NULL for no scope
    description TEXT,                  -- Human-readable description
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure unique permission combinations
    CONSTRAINT unique_permission_combination UNIQUE(resource, action, scope)
);

-- Create indexes for efficient permission lookups
CREATE INDEX idx_permissions_resource ON permissions(resource);
CREATE INDEX idx_permissions_resource_action ON permissions(resource, action);
CREATE INDEX idx_permissions_resource_action_scope ON permissions(resource, action, scope);

-- Roles table: Defines both normal roles and role templates
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_template BOOLEAN NOT NULL DEFAULT false,  -- true for template roles
    is_system BOOLEAN NOT NULL DEFAULT false,    -- true for system-defined roles that cannot be deleted
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for role lookups
CREATE INDEX idx_roles_is_template ON roles(is_template);
CREATE INDEX idx_roles_name ON roles(name);

-- Role permissions: Maps permissions to roles
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (role_id, permission_id)
);

-- Create indexes for efficient permission lookups
CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- User roles: Assigns roles to users (only non-template roles)
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),  -- Who granted this role
    
    PRIMARY KEY (user_id, role_id)
);

-- Create indexes for efficient role lookups
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- User permissions: Custom permissions for individual users
CREATE TABLE user_permissions (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),  -- Who granted this permission
    
    PRIMARY KEY (user_id, permission_id)
);

-- Create indexes for efficient permission lookups
CREATE INDEX idx_user_permissions_user_id ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_permission_id ON user_permissions(permission_id);

-- Function to ensure only non-template roles can be assigned to users
CREATE OR REPLACE FUNCTION check_role_not_template()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM roles 
        WHERE id = NEW.role_id AND is_template = true
    ) THEN
        RAISE EXCEPTION 'Cannot assign template role to user';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to enforce the non-template role constraint
CREATE TRIGGER ensure_non_template_role
    BEFORE INSERT OR UPDATE ON user_roles
    FOR EACH ROW
    EXECUTE FUNCTION check_role_not_template();

-- Add update triggers for tables with updated_at
CREATE TRIGGER update_permissions_updated_at 
    BEFORE UPDATE ON permissions
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roles_updated_at 
    BEFORE UPDATE ON roles
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
