package domain_test

import (
	"testing"

	"backend/internal/authz/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPermission(t *testing.T) {
	tests := []struct {
		name         string
		resource     string
		action       string
		scope        string
		description  string
		wantIDString string
	}{
		{
			name:         "permission without scope",
			resource:     "posts",
			action:       "create",
			scope:        "",
			description:  "Create posts",
			wantIDString: "posts:create",
		},
		{
			name:         "permission with scope",
			resource:     "posts",
			action:       "update",
			scope:        "own",
			description:  "Update own posts",
			wantIDString: "posts:update:own",
		},
		{
			name:         "permission with complex action",
			resource:     "posts",
			action:       "read:draft",
			scope:        "any",
			description:  "Read any draft posts",
			wantIDString: "posts:read:draft:any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := domain.NewPermission(tt.resource, tt.action, tt.scope, tt.description)

			assert.NotNil(t, perm)
			assert.Equal(t, tt.resource, perm.Resource)
			assert.Equal(t, tt.action, perm.Action)
			assert.Equal(t, tt.scope, perm.Scope)
			assert.Equal(t, tt.description, perm.Description)
			assert.Equal(t, tt.wantIDString, perm.IDString())
			assert.NotZero(t, perm.ID)
			assert.NotZero(t, perm.CreatedAt)
			assert.NotZero(t, perm.UpdatedAt)
		})
	}
}

func TestNewPermissionFromID(t *testing.T) {
	tests := []struct {
		name         string
		permissionID string
		description  string
		wantResource string
		wantAction   string
		wantScope    string
		wantErr      bool
	}{
		{
			name:         "simple permission",
			permissionID: "posts:create",
			description:  "Create posts",
			wantResource: "posts",
			wantAction:   "create",
			wantScope:    "",
			wantErr:      false,
		},
		{
			name:         "permission with scope",
			permissionID: "posts:update:own",
			description:  "Update own posts",
			wantResource: "posts",
			wantAction:   "update",
			wantScope:    "own",
			wantErr:      false,
		},
		{
			name:         "permission with complex action and scope",
			permissionID: "posts:read:draft:any",
			description:  "Read any draft posts",
			wantResource: "posts",
			wantAction:   "read:draft",
			wantScope:    "any",
			wantErr:      false,
		},
		{
			name:         "invalid permission - no action",
			permissionID: "posts",
			description:  "Invalid",
			wantErr:      true,
		},
		{
			name:         "empty permission",
			permissionID: "",
			description:  "Invalid",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, err := domain.NewPermissionFromID(tt.permissionID, tt.description)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidPermissionID)
				assert.Nil(t, perm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, perm)
				assert.Equal(t, tt.wantResource, perm.Resource)
				assert.Equal(t, tt.wantAction, perm.Action)
				assert.Equal(t, tt.wantScope, perm.Scope)
				assert.Equal(t, tt.description, perm.Description)
				assert.Equal(t, tt.permissionID, perm.IDString())
			}
		})
	}
}

func TestPermission_IsOwnershipBased(t *testing.T) {
	tests := []struct {
		name       string
		permission *domain.Permission
		want       bool
	}{
		{
			name:       "own scope",
			permission: domain.NewPermission("posts", "update", "own", ""),
			want:       true,
		},
		{
			name:       "self scope",
			permission: domain.NewPermission("users", "read", "self", ""),
			want:       true,
		},
		{
			name:       "any scope",
			permission: domain.NewPermission("posts", "update", "any", ""),
			want:       false,
		},
		{
			name:       "no scope",
			permission: domain.NewPermission("posts", "create", "", ""),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.permission.IsOwnershipBased())
		})
	}
}

func TestPermission_IsGlobal(t *testing.T) {
	tests := []struct {
		name       string
		permission *domain.Permission
		want       bool
	}{
		{
			name:       "any scope",
			permission: domain.NewPermission("posts", "update", "any", ""),
			want:       true,
		},
		{
			name:       "own scope",
			permission: domain.NewPermission("posts", "update", "own", ""),
			want:       false,
		},
		{
			name:       "no scope",
			permission: domain.NewPermission("posts", "create", "", ""),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.permission.IsGlobal())
		})
	}
}

func TestPermission_Matches(t *testing.T) {
	perm := domain.NewPermission("posts", "update", "own", "")

	assert.True(t, perm.Matches("posts", "update"))
	assert.False(t, perm.Matches("posts", "create"))
	assert.False(t, perm.Matches("users", "update"))
	assert.False(t, perm.Matches("users", "create"))
}

func TestParsePermissionID(t *testing.T) {
	tests := []struct {
		name         string
		permissionID string
		wantResource string
		wantAction   string
		wantScope    string
	}{
		{
			name:         "simple permission",
			permissionID: "posts:create",
			wantResource: "posts",
			wantAction:   "create",
			wantScope:    "",
		},
		{
			name:         "permission with scope",
			permissionID: "posts:update:own",
			wantResource: "posts",
			wantAction:   "update",
			wantScope:    "own",
		},
		{
			name:         "permission with complex action",
			permissionID: "posts:read:draft:any",
			wantResource: "posts",
			wantAction:   "read:draft",
			wantScope:    "any",
		},
		{
			name:         "invalid - no colon",
			permissionID: "posts",
			wantResource: "",
			wantAction:   "",
			wantScope:    "",
		},
		{
			name:         "empty string",
			permissionID: "",
			wantResource: "",
			wantAction:   "",
			wantScope:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, action, scope := domain.ParsePermissionID(tt.permissionID)
			assert.Equal(t, tt.wantResource, resource)
			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, tt.wantScope, scope)
		})
	}
}
