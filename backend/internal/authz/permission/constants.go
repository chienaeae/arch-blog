package permission

import "strings"

// Permission represents a structured permission with metadata
type Permission struct {
	ID          string // The permission identifier (e.g., "posts:create")
	Resource    string // The resource being accessed (e.g., "posts")
	Action      string // The action being performed (e.g., "create")
	Scope       string // Optional scope qualifier (e.g., "own", "any", "published")
	Description string // Human-readable description
}

// Permission ID constants
const (
	// Posts permissions
	PostsCreate         = "posts:create"
	PostsReadPublished  = "posts:read:published"
	PostsReadDraftOwn   = "posts:read:draft:own"
	PostsReadDraftAny   = "posts:read:draft:any"
	PostsUpdateOwn      = "posts:update:own"
	PostsUpdateAny      = "posts:update:any"
	PostsDeleteOwn      = "posts:delete:own"
	PostsDeleteAny      = "posts:delete:any"
	PostsPublishOwn     = "posts:publish:own"
	PostsPublishAny     = "posts:publish:any"
	PostsFeature        = "posts:feature"

	// Comments permissions
	CommentsCreate     = "comments:create"
	CommentsRead       = "comments:read"
	CommentsUpdateOwn  = "comments:update:own"
	CommentsUpdateAny  = "comments:update:any"
	CommentsDeleteOwn  = "comments:delete:own"
	CommentsDeleteAny  = "comments:delete:any"
	CommentsModerate   = "comments:moderate"

	// Users permissions
	UsersReadSelf   = "users:read:self"
	UsersReadAny    = "users:read:any"
	UsersUpdateSelf = "users:update:self"
	UsersUpdateAny  = "users:update:any"
	UsersDeleteSelf = "users:delete:self"
	UsersDeleteAny  = "users:delete:any"
	UsersSuspend    = "users:suspend"

	// Media permissions
	MediaUploadOwn = "media:upload:own"
	MediaUploadAny = "media:upload:any"
	MediaReadOwn   = "media:read:own"
	MediaReadAny   = "media:read:any"
	MediaDeleteOwn = "media:delete:own"
	MediaDeleteAny = "media:delete:any"

	// Tags permissions
	TagsCreate = "tags:create"
	TagsRead   = "tags:read"
	TagsUpdate = "tags:update"
	TagsDelete = "tags:delete"

	// Categories permissions
	CategoriesCreate = "categories:create"
	CategoriesRead   = "categories:read"
	CategoriesUpdate = "categories:update"
	CategoriesDelete = "categories:delete"

	// Analytics permissions
	AnalyticsViewOwn   = "analytics:view:own"
	AnalyticsViewAny   = "analytics:view:any"
	AnalyticsExportOwn = "analytics:export:own"
	AnalyticsExportAny = "analytics:export:any"

	// Settings permissions
	SettingsSystem = "settings:system"
	SettingsBlog   = "settings:blog"
	SettingsTheme  = "settings:theme"

	// Authorization permissions (meta permissions)
	AuthzRolesCreate       = "authz:roles:create"
	AuthzRolesRead         = "authz:roles:read"
	AuthzRolesUpdate       = "authz:roles:update"
	AuthzRolesDelete       = "authz:roles:delete"
	AuthzRolesAssign       = "authz:roles:assign"
	AuthzRolesRevoke       = "authz:roles:revoke"
	AuthzPermissionsGrant  = "authz:permissions:grant"
	AuthzPermissionsRevoke = "authz:permissions:revoke"
	AuthzAuditView         = "authz:audit:view"
)

// registry holds all structured Permission objects
var registry = map[string]*Permission{
	// Posts permissions
	PostsCreate:        {ID: PostsCreate, Resource: "posts", Action: "create", Description: "Create new blog posts"},
	PostsReadPublished: {ID: PostsReadPublished, Resource: "posts", Action: "read", Scope: "published", Description: "Read published posts"},
	PostsReadDraftOwn:  {ID: PostsReadDraftOwn, Resource: "posts", Action: "read", Scope: "draft:own", Description: "Read own draft posts"},
	PostsReadDraftAny:  {ID: PostsReadDraftAny, Resource: "posts", Action: "read", Scope: "draft:any", Description: "Read any draft posts"},
	PostsUpdateOwn:     {ID: PostsUpdateOwn, Resource: "posts", Action: "update", Scope: "own", Description: "Update own posts"},
	PostsUpdateAny:     {ID: PostsUpdateAny, Resource: "posts", Action: "update", Scope: "any", Description: "Update any posts"},
	PostsDeleteOwn:     {ID: PostsDeleteOwn, Resource: "posts", Action: "delete", Scope: "own", Description: "Delete own posts"},
	PostsDeleteAny:     {ID: PostsDeleteAny, Resource: "posts", Action: "delete", Scope: "any", Description: "Delete any posts"},
	PostsPublishOwn:    {ID: PostsPublishOwn, Resource: "posts", Action: "publish", Scope: "own", Description: "Publish own posts"},
	PostsPublishAny:    {ID: PostsPublishAny, Resource: "posts", Action: "publish", Scope: "any", Description: "Publish any posts"},
	PostsFeature:       {ID: PostsFeature, Resource: "posts", Action: "feature", Description: "Feature posts on homepage"},

	// Comments permissions
	CommentsCreate:    {ID: CommentsCreate, Resource: "comments", Action: "create", Description: "Create comments"},
	CommentsRead:      {ID: CommentsRead, Resource: "comments", Action: "read", Description: "Read comments"},
	CommentsUpdateOwn: {ID: CommentsUpdateOwn, Resource: "comments", Action: "update", Scope: "own", Description: "Update own comments"},
	CommentsUpdateAny: {ID: CommentsUpdateAny, Resource: "comments", Action: "update", Scope: "any", Description: "Update any comments"},
	CommentsDeleteOwn: {ID: CommentsDeleteOwn, Resource: "comments", Action: "delete", Scope: "own", Description: "Delete own comments"},
	CommentsDeleteAny: {ID: CommentsDeleteAny, Resource: "comments", Action: "delete", Scope: "any", Description: "Delete any comments"},
	CommentsModerate:  {ID: CommentsModerate, Resource: "comments", Action: "moderate", Description: "Moderate comments"},

	// Users permissions
	UsersReadSelf:   {ID: UsersReadSelf, Resource: "users", Action: "read", Scope: "self", Description: "Read own user profile"},
	UsersReadAny:    {ID: UsersReadAny, Resource: "users", Action: "read", Scope: "any", Description: "Read any user profile"},
	UsersUpdateSelf: {ID: UsersUpdateSelf, Resource: "users", Action: "update", Scope: "self", Description: "Update own profile"},
	UsersUpdateAny:  {ID: UsersUpdateAny, Resource: "users", Action: "update", Scope: "any", Description: "Update any user profile"},
	UsersDeleteSelf: {ID: UsersDeleteSelf, Resource: "users", Action: "delete", Scope: "self", Description: "Delete own account"},
	UsersDeleteAny:  {ID: UsersDeleteAny, Resource: "users", Action: "delete", Scope: "any", Description: "Delete any user account"},
	UsersSuspend:    {ID: UsersSuspend, Resource: "users", Action: "suspend", Description: "Suspend user accounts"},

	// Media permissions
	MediaUploadOwn: {ID: MediaUploadOwn, Resource: "media", Action: "upload", Scope: "own", Description: "Upload own media files"},
	MediaUploadAny: {ID: MediaUploadAny, Resource: "media", Action: "upload", Scope: "any", Description: "Upload media for any user"},
	MediaReadOwn:   {ID: MediaReadOwn, Resource: "media", Action: "read", Scope: "own", Description: "Read own media files"},
	MediaReadAny:   {ID: MediaReadAny, Resource: "media", Action: "read", Scope: "any", Description: "Read any media files"},
	MediaDeleteOwn: {ID: MediaDeleteOwn, Resource: "media", Action: "delete", Scope: "own", Description: "Delete own media files"},
	MediaDeleteAny: {ID: MediaDeleteAny, Resource: "media", Action: "delete", Scope: "any", Description: "Delete any media files"},

	// Tags permissions
	TagsCreate: {ID: TagsCreate, Resource: "tags", Action: "create", Description: "Create new tags"},
	TagsRead:   {ID: TagsRead, Resource: "tags", Action: "read", Description: "Read tags"},
	TagsUpdate: {ID: TagsUpdate, Resource: "tags", Action: "update", Description: "Update tags"},
	TagsDelete: {ID: TagsDelete, Resource: "tags", Action: "delete", Description: "Delete tags"},

	// Categories permissions
	CategoriesCreate: {ID: CategoriesCreate, Resource: "categories", Action: "create", Description: "Create categories"},
	CategoriesRead:   {ID: CategoriesRead, Resource: "categories", Action: "read", Description: "Read categories"},
	CategoriesUpdate: {ID: CategoriesUpdate, Resource: "categories", Action: "update", Description: "Update categories"},
	CategoriesDelete: {ID: CategoriesDelete, Resource: "categories", Action: "delete", Description: "Delete categories"},

	// Analytics permissions
	AnalyticsViewOwn:   {ID: AnalyticsViewOwn, Resource: "analytics", Action: "view", Scope: "own", Description: "View own analytics"},
	AnalyticsViewAny:   {ID: AnalyticsViewAny, Resource: "analytics", Action: "view", Scope: "any", Description: "View all analytics"},
	AnalyticsExportOwn: {ID: AnalyticsExportOwn, Resource: "analytics", Action: "export", Scope: "own", Description: "Export own analytics data"},
	AnalyticsExportAny: {ID: AnalyticsExportAny, Resource: "analytics", Action: "export", Scope: "any", Description: "Export all analytics data"},

	// Settings permissions
	SettingsSystem: {ID: SettingsSystem, Resource: "settings", Action: "system", Description: "Manage system settings"},
	SettingsBlog:   {ID: SettingsBlog, Resource: "settings", Action: "blog", Description: "Manage blog settings"},
	SettingsTheme:  {ID: SettingsTheme, Resource: "settings", Action: "theme", Description: "Manage theme settings"},

	// Authorization permissions
	AuthzRolesCreate:       {ID: AuthzRolesCreate, Resource: "authz", Action: "roles:create", Description: "Create roles"},
	AuthzRolesRead:         {ID: AuthzRolesRead, Resource: "authz", Action: "roles:read", Description: "Read roles"},
	AuthzRolesUpdate:       {ID: AuthzRolesUpdate, Resource: "authz", Action: "roles:update", Description: "Update roles"},
	AuthzRolesDelete:       {ID: AuthzRolesDelete, Resource: "authz", Action: "roles:delete", Description: "Delete roles"},
	AuthzRolesAssign:       {ID: AuthzRolesAssign, Resource: "authz", Action: "roles:assign", Description: "Assign roles to users"},
	AuthzRolesRevoke:       {ID: AuthzRolesRevoke, Resource: "authz", Action: "roles:revoke", Description: "Revoke roles from users"},
	AuthzPermissionsGrant:  {ID: AuthzPermissionsGrant, Resource: "authz", Action: "permissions:grant", Description: "Grant permissions"},
	AuthzPermissionsRevoke: {ID: AuthzPermissionsRevoke, Resource: "authz", Action: "permissions:revoke", Description: "Revoke permissions"},
	AuthzAuditView:         {ID: AuthzAuditView, Resource: "authz", Action: "audit:view", Description: "View audit logs"},
}

// FromID looks up a permission by its ID and returns the structured Permission object
func FromID(id string) (*Permission, bool) {
	perm, exists := registry[id]
	return perm, exists
}

// MustFromID looks up a permission by its ID and panics if not found
func MustFromID(id string) *Permission {
	perm, exists := registry[id]
	if !exists {
		panic("permission not found: " + id)
	}
	return perm
}

// All returns all registered permissions
func All() []*Permission {
	result := make([]*Permission, 0, len(registry))
	for _, perm := range registry {
		result = append(result, perm)
	}
	return result
}

// ByResource returns all permissions for a specific resource
func ByResource(resource string) []*Permission {
	var result []*Permission
	for _, perm := range registry {
		if perm.Resource == resource {
			result = append(result, perm)
		}
	}
	return result
}

// IsOwnershipBased returns true if the permission includes ownership scope
func IsOwnershipBased(permissionID string) bool {
	return strings.Contains(permissionID, ":own") || strings.Contains(permissionID, ":self")
}

// IsGlobalPermission returns true if the permission applies to any resource
func IsGlobalPermission(permissionID string) bool {
	return strings.Contains(permissionID, ":any")
}

// IsValid checks if a permission ID exists in the registry
func IsValid(permissionID string) bool {
	_, exists := registry[permissionID]
	return exists
}