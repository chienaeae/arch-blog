package seeder

import "github.com/philly/arch-blog/backend/internal/authz/permission"

// DefaultRole represents a role to be seeded
type DefaultRole struct {
	Name        string
	Description string
	IsTemplate  bool
	IsSystem    bool
}

// DefaultRoles defines all roles to be seeded
var DefaultRoles = []DefaultRole{
	// System roles (cannot be deleted)
	{
		Name:        "super_admin",
		Description: "Full system access with all permissions",
		IsTemplate:  false,
		IsSystem:    true,
	},
	{
		Name:        "admin",
		Description: "Administrative access to manage content and users",
		IsTemplate:  false,
		IsSystem:    true,
	},
	{
		Name:        "editor",
		Description: "Can manage all content but not users",
		IsTemplate:  false,
		IsSystem:    true,
	},
	{
		Name:        "author",
		Description: "Can create and manage own content",
		IsTemplate:  false,
		IsSystem:    true,
	},
	{
		Name:        "contributor",
		Description: "Can create content but cannot publish",
		IsTemplate:  false,
		IsSystem:    true,
	},
	{
		Name:        "subscriber",
		Description: "Can read content and manage own profile",
		IsTemplate:  false,
		IsSystem:    true,
	},
	// Role templates (for creating custom roles)
	{
		Name:        "content_manager_template",
		Description: "Template for content management roles",
		IsTemplate:  true,
		IsSystem:    true,
	},
	{
		Name:        "moderator_template",
		Description: "Template for moderation roles",
		IsTemplate:  true,
		IsSystem:    true,
	},
}

// DefaultRolePermissions defines permission assignments for each role
// Note: super_admin is handled programmatically and gets ALL permissions
var DefaultRolePermissions = map[string][]string{
	"admin": {
		// Admin can manage content and users but not system settings
		permission.PostsCreate, permission.PostsReadPublished, permission.PostsReadDraftAny,
		permission.PostsUpdateAny, permission.PostsDeleteAny, permission.PostsPublishAny, permission.PostsFeature,
		permission.CommentsCreate, permission.CommentsRead, permission.CommentsUpdateAny,
		permission.CommentsDeleteAny, permission.CommentsModerate,
		permission.UsersReadAny, permission.UsersUpdateAny, permission.UsersSuspend,
		permission.MediaUploadAny, permission.MediaReadAny, permission.MediaDeleteAny,
		permission.TagsCreate, permission.TagsRead, permission.TagsUpdate, permission.TagsDelete,
		permission.CategoriesCreate, permission.CategoriesRead, permission.CategoriesUpdate, permission.CategoriesDelete,
		permission.AnalyticsViewAny, permission.AnalyticsExportAny,
		permission.SettingsBlog, permission.SettingsTheme,
		permission.AuthzRolesRead, permission.AuthzRolesAssign, permission.AuthzRolesRevoke,
		permission.AuthzAuditView,
	},
	"editor": {
		// Editor can manage all content but not users
		permission.PostsCreate, permission.PostsReadPublished, permission.PostsReadDraftAny,
		permission.PostsUpdateAny, permission.PostsDeleteAny, permission.PostsPublishAny, permission.PostsFeature,
		permission.CommentsCreate, permission.CommentsRead, permission.CommentsUpdateAny,
		permission.CommentsDeleteAny, permission.CommentsModerate,
		permission.UsersReadSelf, permission.UsersUpdateSelf,
		permission.MediaUploadAny, permission.MediaReadAny, permission.MediaDeleteAny,
		permission.TagsCreate, permission.TagsRead, permission.TagsUpdate, permission.TagsDelete,
		permission.CategoriesCreate, permission.CategoriesRead, permission.CategoriesUpdate, permission.CategoriesDelete,
		permission.AnalyticsViewAny, permission.AnalyticsExportAny,
	},
	"author": {
		// Author can create and manage own content
		permission.PostsCreate, permission.PostsReadPublished, permission.PostsReadDraftOwn,
		permission.PostsUpdateOwn, permission.PostsDeleteOwn, permission.PostsPublishOwn,
		permission.CommentsCreate, permission.CommentsRead, permission.CommentsUpdateOwn, permission.CommentsDeleteOwn,
		permission.UsersReadSelf, permission.UsersUpdateSelf,
		permission.MediaUploadOwn, permission.MediaReadOwn, permission.MediaDeleteOwn,
		permission.TagsRead, permission.CategoriesRead,
		permission.AnalyticsViewOwn, permission.AnalyticsExportOwn,
	},
	"contributor": {
		// Contributor can create content but cannot publish
		permission.PostsCreate, permission.PostsReadPublished, permission.PostsReadDraftOwn,
		permission.PostsUpdateOwn, permission.PostsDeleteOwn,
		permission.CommentsCreate, permission.CommentsRead, permission.CommentsUpdateOwn, permission.CommentsDeleteOwn,
		permission.UsersReadSelf, permission.UsersUpdateSelf,
		permission.MediaUploadOwn, permission.MediaReadOwn,
		permission.TagsRead, permission.CategoriesRead,
		permission.AnalyticsViewOwn,
	},
	"subscriber": {
		// Subscriber can read content and manage own profile
		permission.PostsReadPublished,
		permission.CommentsCreate, permission.CommentsRead, permission.CommentsUpdateOwn, permission.CommentsDeleteOwn,
		permission.UsersReadSelf, permission.UsersUpdateSelf,
		permission.TagsRead, permission.CategoriesRead,
	},
	"content_manager_template": {
		// Template with content management permissions
		permission.PostsCreate, permission.PostsReadDraftAny, permission.PostsUpdateAny,
		permission.PostsPublishAny, permission.PostsFeature,
		permission.MediaUploadAny, permission.MediaReadAny,
		permission.TagsCreate, permission.TagsUpdate,
		permission.CategoriesCreate, permission.CategoriesUpdate,
	},
	"moderator_template": {
		// Template with moderation permissions
		permission.CommentsRead, permission.CommentsUpdateAny, permission.CommentsDeleteAny,
		permission.CommentsModerate,
		permission.PostsReadDraftAny,
		permission.UsersReadAny, permission.UsersSuspend,
	},
}
