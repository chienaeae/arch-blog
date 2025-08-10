package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRole(t *testing.T) {
	role := domain.NewRole("editor", "Can edit content")
	
	assert.NotNil(t, role)
	assert.NotEqual(t, uuid.Nil, role.ID)
	assert.Equal(t, "editor", role.Name)
	assert.Equal(t, "Can edit content", role.Description)
	assert.False(t, role.IsTemplate)
	assert.False(t, role.IsSystem)
	assert.Empty(t, role.Permissions)
	assert.NotZero(t, role.CreatedAt)
	assert.NotZero(t, role.UpdatedAt)
}

func TestNewSystemRole(t *testing.T) {
	role := domain.NewSystemRole("admin", "Administrator")
	
	assert.NotNil(t, role)
	assert.Equal(t, "admin", role.Name)
	assert.False(t, role.IsTemplate)
	assert.True(t, role.IsSystem)
}

func TestNewTemplateRole(t *testing.T) {
	role := domain.NewTemplateRole("content_template", "Template for content roles")
	
	assert.NotNil(t, role)
	assert.Equal(t, "content_template", role.Name)
	assert.True(t, role.IsTemplate)
	assert.True(t, role.IsSystem) // Templates are also system roles
}

func TestRole_AddPermission(t *testing.T) {
	role := domain.NewRole("editor", "Can edit content")
	perm1 := domain.NewPermission("posts", "create", "", "")
	perm2 := domain.NewPermission("posts", "update", "own", "")
	
	// Add first permission
	err := role.AddPermission(perm1)
	require.NoError(t, err)
	assert.Len(t, role.Permissions, 1)
	assert.Equal(t, perm1.ID, role.Permissions[0].ID)
	
	// Add second permission
	err = role.AddPermission(perm2)
	require.NoError(t, err)
	assert.Len(t, role.Permissions, 2)
	
	// Try to add duplicate permission
	err = role.AddPermission(perm1)
	assert.ErrorIs(t, err, domain.ErrPermissionExists)
	assert.Len(t, role.Permissions, 2)
	
	// Try to add nil permission
	err = role.AddPermission(nil)
	assert.ErrorIs(t, err, domain.ErrPermissionNil)
	assert.Len(t, role.Permissions, 2)
}

func TestRole_RemovePermission(t *testing.T) {
	role := domain.NewRole("editor", "Can edit content")
	perm1 := domain.NewPermission("posts", "create", "", "")
	perm2 := domain.NewPermission("posts", "update", "own", "")
	
	// Add permissions
	_ = role.AddPermission(perm1)
	_ = role.AddPermission(perm2)
	require.Len(t, role.Permissions, 2)
	
	// Remove first permission
	err := role.RemovePermission(perm1.ID)
	require.NoError(t, err)
	assert.Len(t, role.Permissions, 1)
	assert.Equal(t, perm2.ID, role.Permissions[0].ID)
	
	// Try to remove non-existent permission
	err = role.RemovePermission(uuid.New())
	assert.ErrorIs(t, err, domain.ErrPermissionNotFound)
	assert.Len(t, role.Permissions, 1)
}

func TestRole_HasPermission(t *testing.T) {
	role := domain.NewRole("editor", "Can edit content")
	perm1 := domain.NewPermission("posts", "create", "", "")
	perm2 := domain.NewPermission("posts", "update", "own", "")
	
	_ = role.AddPermission(perm1)
	_ = role.AddPermission(perm2)
	
	assert.True(t, role.HasPermission("posts:create"))
	assert.True(t, role.HasPermission("posts:update:own"))
	assert.False(t, role.HasPermission("posts:delete"))
	assert.False(t, role.HasPermission("users:create"))
}

func TestRole_HasPermissionForResource(t *testing.T) {
	role := domain.NewRole("editor", "Can edit content")
	perm1 := domain.NewPermission("posts", "create", "", "")
	perm2 := domain.NewPermission("posts", "update", "own", "")
	perm3 := domain.NewPermission("users", "read", "any", "")
	
	_ = role.AddPermission(perm1)
	_ = role.AddPermission(perm2)
	_ = role.AddPermission(perm3)
	
	assert.True(t, role.HasPermissionForResource("posts", "create"))
	assert.True(t, role.HasPermissionForResource("posts", "update"))
	assert.True(t, role.HasPermissionForResource("users", "read"))
	assert.False(t, role.HasPermissionForResource("posts", "delete"))
	assert.False(t, role.HasPermissionForResource("users", "update"))
}

func TestRole_CanBeAssigned(t *testing.T) {
	normalRole := domain.NewRole("editor", "")
	templateRole := domain.NewTemplateRole("template", "")
	
	assert.True(t, normalRole.CanBeAssigned())
	assert.False(t, templateRole.CanBeAssigned())
}

func TestRole_CanBeDeleted(t *testing.T) {
	normalRole := domain.NewRole("custom", "")
	systemRole := domain.NewSystemRole("admin", "")
	templateRole := domain.NewTemplateRole("template", "")
	
	assert.True(t, normalRole.CanBeDeleted())
	assert.False(t, systemRole.CanBeDeleted())
	assert.False(t, templateRole.CanBeDeleted()) // Templates are system roles
}

func TestRole_Validate(t *testing.T) {
	normalRole := domain.NewRole("editor", "")
	templateRole := domain.NewTemplateRole("template", "")
	
	err := normalRole.Validate()
	assert.NoError(t, err)
	
	err = templateRole.Validate()
	assert.ErrorIs(t, err, domain.ErrTemplateCannotAssign)
}

func TestRole_ValidateDeletion(t *testing.T) {
	normalRole := domain.NewRole("custom", "")
	systemRole := domain.NewSystemRole("admin", "")
	
	err := normalRole.ValidateDeletion()
	assert.NoError(t, err)
	
	err = systemRole.ValidateDeletion()
	assert.ErrorIs(t, err, domain.ErrSystemCannotDelete)
}

func TestRole_CloneAsCustomRole(t *testing.T) {
	// Create a template role with permissions
	template := domain.NewTemplateRole("content_template", "Template for content roles")
	perm1 := domain.NewPermission("posts", "create", "", "")
	perm2 := domain.NewPermission("posts", "update", "own", "")
	_ = template.AddPermission(perm1)
	_ = template.AddPermission(perm2)
	
	// Clone the template
	cloned, err := template.CloneAsCustomRole("custom_editor", "Custom editor role")
	require.NoError(t, err)
	require.NotNil(t, cloned)
	
	assert.Equal(t, "custom_editor", cloned.Name)
	assert.Equal(t, "Custom editor role", cloned.Description)
	assert.False(t, cloned.IsTemplate)
	assert.False(t, cloned.IsSystem)
	assert.Len(t, cloned.Permissions, 2)
	assert.Equal(t, perm1.ID, cloned.Permissions[0].ID)
	assert.Equal(t, perm2.ID, cloned.Permissions[1].ID)
	
	// Try to clone a non-template role
	normalRole := domain.NewRole("editor", "")
	_, err = normalRole.CloneAsCustomRole("new", "")
	assert.ErrorIs(t, err, domain.ErrOnlyTemplateCanClone)
}