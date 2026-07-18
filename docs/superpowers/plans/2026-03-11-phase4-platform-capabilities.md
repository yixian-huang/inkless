# Phase 4 Implementation Plan: Platform Capabilities (平台能力)

> 历史设计（2026-07-18 标记）：本计划记录早期共享数据库多站点设想，不再代表当前 Roadmap 或产品承诺。当前决策是一个实例服务一个逻辑站点，多个独立站点部署多个实例；不要据此为核心内容模型增加 `site_id`。参见 `docs/adr/0001-single-instance-single-site.md`。

**Date**: 2026-03-11
**Design Spec**: `docs/superpowers/specs/2026-03-11-open-source-evolution-design.md`
**Prerequisites**: Phase 0-3 complete (migrations via goose, provider registry, event bus, plugin architecture)
**Estimated Duration**: 3-4 weeks
**Approach**: TDD — write tests first, then implement

---

## Table of Contents

- [4.1 RBAC Permission System](#41-rbac-permission-system)
- [4.2 Multi-site Support](#42-multi-site-support)
- [4.3 Storage & Media Enhancement](#43-storage--media-enhancement)
- [4.4 Data Migration](#44-data-migration)
- [4.5 Operations & Monitoring](#45-operations--monitoring)
- [Verification Commands](#verification-commands)
- [Migration Safety Checklist](#migration-safety-checklist)

---

## 4.1 RBAC Permission System

**Current State**: The system uses a simple `Role` enum (`admin`/`editor`) on the `User` model with `IsSuperAdmin` boolean. Permissions are stored as a JSON `StringSlice` on the user. Middleware checks are hardcoded: `RequireAdmin()`, `RequireAdminOrEditor()`, `RequireSuperAdmin()`. JWT claims carry `role` as a string.

**Goal**: Replace the flat role system with a proper RBAC model supporting custom roles, granular permissions, and extensibility for plugins and multi-site.

---

### 4.1.1 RBAC Model Design (0.5d)

**New files:**
- `backend/internal/model/role.go`
- `backend/internal/model/permission.go`
- `backend/internal/model/role_test.go`
- `backend/internal/model/permission_test.go`
- `backend/internal/db/migrations/YYYYMMDDHHMMSS_create_rbac_tables.sql`

**Database schema (goose migration):**

```sql
-- +goose Up
CREATE TABLE roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    site_id     INTEGER,  -- NULL = global role, set = site-scoped (Phase 4.2)
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    resource    VARCHAR(50) NOT NULL,   -- e.g. "articles", "pages", "media"
    action      VARCHAR(50) NOT NULL,   -- e.g. "create", "read", "update", "delete", "publish"
    description TEXT,
    UNIQUE(resource, action)
);

CREATE TABLE role_permissions (
    role_id       INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id   INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    site_id   INTEGER,  -- NULL = global assignment (Phase 4.2 activates this)
    PRIMARY KEY (user_id, role_id, COALESCE(site_id, 0))
);

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
```

**Model definitions:**

```go
// backend/internal/model/role.go
package model

import "time"

type Role struct {
    ID          uint         `gorm:"primaryKey" json:"id"`
    Name        string       `gorm:"uniqueIndex;not null;size:50" json:"name"`
    DisplayName string       `gorm:"not null;size:100" json:"displayName"`
    Description string       `gorm:"type:text" json:"description"`
    IsSystem    bool         `gorm:"not null;default:false" json:"isSystem"`
    SiteID      *uint        `gorm:"index" json:"siteId"`
    Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
    CreatedAt   time.Time    `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
}

type UserRole struct {
    UserID uint  `gorm:"primaryKey" json:"userId"`
    RoleID uint  `gorm:"primaryKey" json:"roleId"`
    SiteID *uint `json:"siteId"` // NULL = global
    Role   Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}
```

```go
// backend/internal/model/permission.go
package model

type Permission struct {
    ID          uint   `gorm:"primaryKey" json:"id"`
    Resource    string `gorm:"not null;size:50;uniqueIndex:idx_resource_action" json:"resource"`
    Action      string `gorm:"not null;size:50;uniqueIndex:idx_resource_action" json:"action"`
    Description string `gorm:"type:text" json:"description"`
}
```

**Refactor existing `model/user.go`:**
- Keep the `Role` string field temporarily for backward compatibility during migration
- Add `Roles []UserRole` relationship
- Deprecate `Permissions StringSlice` field (keep column, stop writing)
- Add helper: `func (u *User) HasRBACPermission(resource, action string) bool`

**Test (write first):**

```go
// backend/internal/model/role_test.go
func TestRole_Validate(t *testing.T) {
    // Name required, DisplayName required, Name format (alphanumeric + hyphens)
}

func TestPermission_Key(t *testing.T) {
    p := Permission{Resource: "articles", Action: "create"}
    assert.Equal(t, "articles:create", p.Key())
}
```

**Tasks:**
1. Write model tests for Role, Permission, UserRole validation
2. Create goose migration SQL
3. Implement model structs
4. Run `cd backend && go test -v -race ./internal/model/...`

---

### 4.1.2 Built-in Roles & Permission Seed (0.5d)

**Modified files:**
- `backend/internal/seed/rbac_seed.go`
- `backend/internal/seed/rbac_seed_test.go`

**Built-in roles and their permissions:**

| Role | Permissions |
|------|-------------|
| `super_admin` | `*:*` (all resources, all actions) — IsSystem=true |
| `site_admin` | All except `system:*`, `users:delete` |
| `editor` | `articles:*`, `pages:*`, `media:*`, `comments:*`, `categories:*`, `tags:*`, `menus:read` |
| `author` | `articles:create`, `articles:read`, `articles:update` (own only), `media:create`, `media:read`, `comments:read` |
| `viewer` | `*:read` (read-only access to all resources) |

**Permission resources** (superset of existing `ValidPermissions`):

```go
var BuiltinResources = []string{
    "dashboard", "articles", "pages", "media", "comments",
    "categories", "tags", "menus", "themes", "analytics",
    "audit_logs", "backups", "users", "form_submissions",
    "roles", "settings", "system", "sites", "plugins",
}

var BuiltinActions = []string{
    "create", "read", "update", "delete", "publish", "manage",
}
```

**Seed logic:** Idempotent — check if role exists before creating. Never modify user-created roles.

**Tasks:**
1. Write seed test: assert all 5 roles created with correct permission counts
2. Implement `SeedRBAC(ctx context.Context, db *gorm.DB) error`
3. Wire into `seed.NewSeeder()` call chain in `cmd/server/main.go`
4. Run `cd backend && go test -v -race ./internal/seed/...`

---

### 4.1.3 Permission Middleware Refactor (0.5d)

**Modified files:**
- `backend/internal/middleware/auth.go`
- `backend/internal/middleware/rbac.go` (new)
- `backend/internal/middleware/rbac_test.go` (new)
- `backend/internal/middleware/auth_test.go`

**New middleware:**

```go
// backend/internal/middleware/rbac.go
package middleware

// RequirePermission returns middleware that checks if the authenticated user
// has a specific permission (resource:action) via their assigned roles.
// Falls back to legacy Role field check for backward compatibility.
func RequirePermission(resource, action string, userRepo repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        userCtx := GetUserContext(c)
        if userCtx == nil {
            respondWithError(c, apierror.Unauthorized("Authentication required"))
            return
        }

        // Load user with roles and permissions
        user, err := userRepo.FindByIDWithRoles(c.Request.Context(), userCtx.UserID)
        if err != nil {
            respondWithError(c, apierror.Forbidden("User not found"))
            return
        }

        if !user.HasRBACPermission(resource, action) {
            respondWithError(c, apierror.Forbidden(
                fmt.Sprintf("Permission denied: %s:%s", resource, action),
            ))
            return
        }

        // Inject full user into context for downstream handlers
        c.Set("rbac_user", user)
        c.Next()
    }
}

// RequireAnyPermission checks if user has at least one of the given permissions.
func RequireAnyPermission(perms []struct{ Resource, Action string }, userRepo repository.UserRepository) gin.HandlerFunc
```

**Migration strategy for existing routes:**
- Keep `RequireAdmin()`, `RequireAdminOrEditor()`, `RequireSuperAdmin()` working (they check legacy `Role` field)
- Gradually replace in route registration — one handler group at a time
- Add `UserContext.Permissions []string` for caching loaded permissions per request

**Route migration example (in `cmd/server/main.go`):**

```go
// Before:
adminPublish.Use(middleware.RequireAdmin())

// After:
adminPublish.Use(middleware.RequirePermission("articles", "publish", userRepo))
```

**UserContext enhancement:**

```go
type UserContext struct {
    UserID      uint
    Username    string
    Role        model.Role          // legacy, kept for compatibility
    Permissions map[string]struct{} // loaded from RBAC, cached per-request
    IsSuperAdmin bool
}
```

**Test (write first):**

```go
// backend/internal/middleware/rbac_test.go
func TestRequirePermission_AllowsWithPermission(t *testing.T) {
    // Setup: user with editor role -> has articles:create
    // Assert: request passes through
}

func TestRequirePermission_DeniesWithoutPermission(t *testing.T) {
    // Setup: user with author role -> no users:delete
    // Assert: 403 response
}

func TestRequirePermission_SuperAdminBypassesAll(t *testing.T) {
    // Setup: super admin user
    // Assert: any permission check passes
}
```

**Tasks:**
1. Write middleware tests
2. Implement `RequirePermission` and `RequireAnyPermission`
3. Add `FindByIDWithRoles` to UserRepository interface and impl
4. Update `UserContext` struct
5. Replace middleware usage in `cmd/server/main.go` route-by-route
6. Run `cd backend && go test -v -race ./internal/middleware/...`
7. Run `cd backend && go vet ./...`

---

### 4.1.4 Permission Management UI (0.5d)

**New files:**
- `frontend/src/pages/admin/roles/page.tsx` — Role list page
- `frontend/src/pages/admin/roles/RoleEditor.tsx` — Create/edit role dialog
- `frontend/src/pages/admin/roles/PermissionMatrix.tsx` — Resource x Action checkbox grid
- `frontend/src/api/roles.ts` — API client
- `frontend/src/pages/admin/roles/page.test.tsx`

**New API endpoints (add to `cmd/server/main.go`):**

```
GET    /admin/roles              — list all roles
GET    /admin/roles/:id          — get role with permissions
POST   /admin/roles              — create custom role
PUT    /admin/roles/:id          — update role (system roles: only permissions editable)
DELETE /admin/roles/:id          — delete role (reject if system or has users)
GET    /admin/permissions        — list all available permissions
```

**Backend handler:**
- `backend/internal/handler/role/handler.go`
- `backend/internal/handler/role/handler_test.go`

**Backend repository:**
- `backend/internal/repository/role_repository.go` — interface
- `backend/internal/repository/role_repository_impl.go` — GORM impl

**Repository interface:**

```go
type RoleRepository interface {
    Create(ctx context.Context, role *model.Role) error
    FindByID(ctx context.Context, id uint) (*model.Role, error)
    FindByName(ctx context.Context, name string) (*model.Role, error)
    Update(ctx context.Context, role *model.Role) error
    Delete(ctx context.Context, id uint) error
    List(ctx context.Context) ([]*model.Role, error)
    SetPermissions(ctx context.Context, roleID uint, permissionIDs []uint) error
    ListPermissions(ctx context.Context) ([]*model.Permission, error)
    CountUsersWithRole(ctx context.Context, roleID uint) (int64, error)
}
```

**Frontend `PermissionMatrix` component:**

A grid with resources as rows and actions as columns. Checkboxes toggle permissions. System roles show read-only checkboxes.

```tsx
// frontend/src/pages/admin/roles/PermissionMatrix.tsx
interface Props {
  permissions: Permission[];
  selected: Set<number>;
  onChange: (permId: number, checked: boolean) => void;
  readonly?: boolean;
}
```

**Frontend test:**

```tsx
// frontend/src/pages/admin/roles/page.test.tsx
test("renders role list", async () => { /* mock GET /admin/roles */ });
test("opens create dialog", async () => { /* click button, assert form */ });
test("permission matrix toggles", async () => { /* click checkbox */ });
```

**Tasks:**
1. Write handler tests (Go)
2. Implement RoleRepository interface and GORM impl
3. Implement role handler
4. Wire routes in `cmd/server/main.go`
5. Add route to `frontend/src/router/config.tsx`
6. Write frontend component tests
7. Build PermissionMatrix, RoleEditor, page components
8. Run `pnpm lint && pnpm type-check && pnpm test`
9. Run `cd backend && go test -v -race ./...`

---

### 4.1.5 User Registration & Invite (0.5d)

**New/modified files:**
- `backend/internal/handler/auth/register.go`
- `backend/internal/handler/auth/register_test.go`
- `backend/internal/model/invite.go`
- `backend/internal/repository/invite_repository.go`
- `backend/internal/repository/invite_repository_impl.go`
- `backend/internal/db/migrations/YYYYMMDDHHMMSS_create_invites_table.sql`
- `frontend/src/pages/register/page.tsx`
- `frontend/src/pages/admin/users/InviteDialog.tsx`

**Invite model:**

```go
// backend/internal/model/invite.go
type Invite struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Code      string    `gorm:"uniqueIndex;not null;size:64" json:"code"`
    Email     string    `gorm:"size:255" json:"email"`        // optional: restrict to specific email
    RoleID    uint      `gorm:"not null" json:"roleId"`       // assigned role on registration
    SiteID    *uint     `gorm:"index" json:"siteId"`          // Phase 4.2
    MaxUses   int       `gorm:"default:1" json:"maxUses"`
    UsedCount int       `gorm:"default:0" json:"usedCount"`
    ExpiresAt time.Time `json:"expiresAt"`
    CreatedBy uint      `json:"createdBy"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
```

**System setting (stored in site config or env):**

```go
// Registration mode: "disabled" | "invite_only" | "open"
// Default: "invite_only"
REGISTRATION_MODE=invite_only
```

**API endpoints:**

```
POST /auth/register               — public registration (checks mode + optional invite code)
POST /admin/invites               — create invite code (admin+)
GET  /admin/invites               — list invites
DELETE /admin/invites/:id         — revoke invite
```

**Registration flow:**
1. Check `REGISTRATION_MODE` setting
2. If `disabled`: reject
3. If `invite_only`: require valid, unexpired, under-max-uses invite code
4. If `open`: allow (assign default `viewer` role)
5. Create user, assign role from invite (or default), return token pair

**Test:**

```go
func TestRegister_InviteOnly_ValidCode(t *testing.T) {
    // Create invite, register with code, assert user created with correct role
}
func TestRegister_Disabled_Rejects(t *testing.T) {}
func TestRegister_Open_AssignsDefaultRole(t *testing.T) {}
func TestRegister_ExpiredInvite_Rejects(t *testing.T) {}
```

**Tasks:**
1. Write registration handler tests
2. Create invite model and migration
3. Implement invite repository
4. Implement registration handler
5. Add `/auth/register` route
6. Build frontend registration page
7. Build invite management dialog in admin users page
8. Run full test suite

---

### 4.1.6 OAuth2 Provider Interface (0.5d)

**New files:**
- `backend/internal/provider/oauth2.go`
- `backend/internal/provider/oauth2_test.go`

**Interface definition only** (implementations left to plugins):

```go
// backend/internal/provider/oauth2.go
package provider

import "context"

// OAuth2UserInfo represents the user information returned by an OAuth2 provider.
type OAuth2UserInfo struct {
    ProviderID   string // e.g. "github", "google", "wechat"
    ExternalID   string // provider-specific user ID
    Email        string
    DisplayName  string
    AvatarURL    string
    RawData      map[string]interface{} // full provider response
}

// OAuth2Provider defines the contract for OAuth2 authentication providers.
// Plugins implement this interface for GitHub, Google, WeChat, etc.
type OAuth2Provider interface {
    // ID returns the unique provider identifier (e.g. "github").
    ID() string

    // AuthURL returns the URL to redirect the user to for authentication.
    AuthURL(state string) string

    // Exchange exchanges an authorization code for user information.
    Exchange(ctx context.Context, code string) (*OAuth2UserInfo, error)
}
```

**User model extension (migration):**

```sql
-- +goose Up
CREATE TABLE user_oauth_links (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    email       VARCHAR(255),
    avatar_url  VARCHAR(500),
    linked_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, external_id)
);
CREATE INDEX idx_oauth_user ON user_oauth_links(user_id);

-- +goose Down
DROP TABLE IF EXISTS user_oauth_links;
```

**Auth handler extension (stub routes):**

```
GET  /auth/oauth/:provider        — redirect to provider AuthURL
GET  /auth/oauth/:provider/callback — handle callback, link/create user, return tokens
GET  /admin/me/oauth-links        — list linked providers
DELETE /admin/me/oauth-links/:provider — unlink provider
```

**Tasks:**
1. Write interface and types
2. Create migration for `user_oauth_links`
3. Register "oauth2" provider type in registry
4. Add stub handler that returns 501 if no provider registered
5. Write test for stub behavior
6. Run `cd backend && go test -v -race ./internal/provider/...`

---

### 4.1.7 Profile Page (0.5d)

**New/modified files:**
- `frontend/src/pages/admin/profile/page.tsx`
- `frontend/src/pages/admin/profile/page.test.tsx`
- `frontend/src/api/profile.ts`
- `backend/internal/handler/auth/profile.go`
- `backend/internal/handler/auth/profile_test.go`

**User model extension (add fields via migration):**

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN display_name VARCHAR(100);
ALTER TABLE users ADD COLUMN email VARCHAR(255);
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);
ALTER TABLE users ADD COLUMN last_login_at DATETIME;
ALTER TABLE users ADD COLUMN last_login_ip VARCHAR(45);

-- +goose Down
-- SQLite doesn't support DROP COLUMN easily; these are additive-only
```

**API endpoints:**

```
GET  /admin/me/profile            — get current user profile (includes roles, permissions)
PUT  /admin/me/profile            — update display_name, email, avatar
PUT  /admin/me/password           — change password (requires current password)
GET  /admin/me/sessions           — list active refresh tokens (login devices)
DELETE /admin/me/sessions/:id     — revoke a specific session
```

**Profile page features:**
- Avatar upload (reuse media upload endpoint)
- Display name and email edit
- Password change form (current + new + confirm)
- Active sessions list with IP/device info and revoke button
- Linked OAuth providers (from 4.1.6, read-only until plugins exist)

**Frontend test:**

```tsx
test("renders profile with user data", async () => {});
test("updates display name", async () => {});
test("password change requires current password", async () => {});
test("revokes session", async () => {});
```

**Tasks:**
1. Write handler tests (Go)
2. Create user field migration
3. Implement profile handler
4. Wire routes
5. Write frontend tests
6. Build profile page component
7. Add route to `frontend/src/router/config.tsx`
8. Run `pnpm lint && pnpm type-check && pnpm test`

---

## 4.2 Multi-site Support

**Current State**: All content tables are global (no site scoping). A single installation serves one site. The `Comment` model already has a `SiteID *uint` field (forward-looking).

**Goal**: Support multiple sites from a single installation, each with independent content, config, themes, and user roles.

---

### 4.2.1 Site Model Design (0.5d)

**New files:**
- `backend/internal/model/site.go`
- `backend/internal/model/site_test.go`
- `backend/internal/db/migrations/YYYYMMDDHHMMSS_create_sites_table.sql`

**Schema:**

```go
// backend/internal/model/site.go
type Site struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Name        string    `gorm:"not null;size:100" json:"name"`
    Slug        string    `gorm:"uniqueIndex;not null;size:50" json:"slug"`
    Domain      string    `gorm:"size:255;index" json:"domain"`      // e.g. "blog.example.com"
    BasePath    string    `gorm:"size:100" json:"basePath"`           // e.g. "/blog" for subpath mode
    Locale      string    `gorm:"size:10;default:'zh'" json:"locale"` // default locale
    ThemeID     string    `gorm:"size:100" json:"themeId"`
    Config      JSONMap   `gorm:"type:jsonb" json:"config"`           // site-level settings
    IsDefault   bool      `gorm:"default:false" json:"isDefault"`     // exactly one default site
    IsActive    bool      `gorm:"default:true" json:"isActive"`
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
```

**Migration creates a default site and assigns existing data to it:**

```sql
-- +goose Up
CREATE TABLE sites (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(50) NOT NULL UNIQUE,
    domain     VARCHAR(255),
    base_path  VARCHAR(100),
    locale     VARCHAR(10) NOT NULL DEFAULT 'zh',
    theme_id   VARCHAR(100),
    config     TEXT, -- JSON
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create the default site from existing data
INSERT INTO sites (name, slug, is_default, is_active)
VALUES ('Default Site', 'default', TRUE, TRUE);

-- +goose Down
DROP TABLE IF EXISTS sites;
```

**Tasks:**
1. Write model tests
2. Create migration
3. Implement Site model
4. Run `cd backend && go test -v -race ./internal/model/...`

---

### 4.2.2 Data Isolation with site_id (1d)

**Modified files (migrations):**
- `backend/internal/db/migrations/YYYYMMDDHHMMSS_add_site_id_to_content_tables.sql`

**Tables requiring `site_id` column:**

| Table | Current `site_id`? | Action |
|-------|--------------------|--------|
| `articles` | No | Add nullable `site_id`, backfill with default site |
| `pages` | No | Add nullable `site_id`, backfill |
| `content_documents` | No | Add nullable `site_id`, backfill |
| `media` | No | Add nullable `site_id`, backfill |
| `categories` | No | Add nullable `site_id`, backfill |
| `tags` | No | Add nullable `site_id`, backfill |
| `comments` | Yes (`*uint`) | Backfill with default site |
| `menus` (menu_groups) | No | Add nullable `site_id`, backfill |
| `form_submissions` | No | Add nullable `site_id`, backfill |
| `page_views` | No | Add nullable `site_id`, backfill |
| `search_index_entries` | No | Add nullable `site_id`, backfill |
| `audit_events` | No | Add nullable `site_id` (for filtering, not isolation) |
| `backup_records` | No | Add nullable `site_id` |
| `installed_themes` | No | Add nullable `site_id`, backfill |

**Migration strategy:**

```sql
-- +goose Up
-- Add site_id to all content tables
ALTER TABLE articles ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE pages ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE content_documents ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE media ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE categories ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE tags ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE menu_groups ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE form_submissions ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE page_views ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE search_index_entries ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE audit_events ADD COLUMN site_id INTEGER;
ALTER TABLE backup_records ADD COLUMN site_id INTEGER REFERENCES sites(id);
ALTER TABLE installed_themes ADD COLUMN site_id INTEGER REFERENCES sites(id);

-- Backfill existing data with default site ID
UPDATE articles SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE pages SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE content_documents SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE media SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE categories SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE tags SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE comments SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE menu_groups SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE form_submissions SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE page_views SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE search_index_entries SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);
UPDATE installed_themes SET site_id = (SELECT id FROM sites WHERE is_default = TRUE);

-- Add indexes
CREATE INDEX idx_articles_site ON articles(site_id);
CREATE INDEX idx_pages_site ON pages(site_id);
CREATE INDEX idx_media_site ON media(site_id);
CREATE INDEX idx_categories_site ON categories(site_id);
CREATE INDEX idx_tags_site ON tags(site_id);

-- +goose Down
-- SQLite: cannot drop columns; for PostgreSQL:
-- ALTER TABLE articles DROP COLUMN site_id; (etc.)
```

**GORM scope for automatic site filtering:**

```go
// backend/internal/db/site_scope.go
package db

import "gorm.io/gorm"

// SiteScope returns a GORM scope that filters by site_id.
// Use with db.Scopes(SiteScope(siteID)) on all queries.
func SiteScope(siteID uint) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("site_id = ?", siteID)
    }
}

// WithSiteID sets site_id on a model before create.
func WithSiteID(siteID uint) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Set("site_id", siteID)
    }
}
```

**Repository refactoring strategy:**
- All `_repository_impl.go` files get an optional `siteID *uint` field
- When `siteID != nil`, all queries add `.Scopes(db.SiteScope(*siteID))`
- Factory functions accept optional site ID: `NewGormArticleRepository(db, siteID)`
- Single-site deployments pass `nil` (no filtering, backward compatible)

**Tasks:**
1. Write migration SQL
2. Write `SiteScope` tests
3. Implement `SiteScope`
4. Refactor one repository as proof-of-concept (ArticleRepository)
5. Extend to all repositories
6. Update all model structs to include `SiteID *uint` field
7. Run `cd backend && go test -v -race ./...`

---

### 4.2.3 Site Context Middleware (0.5d)

**New files:**
- `backend/internal/middleware/site.go`
- `backend/internal/middleware/site_test.go`

**Implementation:**

```go
// backend/internal/middleware/site.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "blotting-consultancy/internal/repository"
)

const SiteContextKey = "current_site"

// SiteContext resolves the current site from the request and injects it into context.
// Resolution order:
//   1. X-Site-ID header (admin API calls)
//   2. Host header -> match site domain
//   3. URL path prefix -> match site base_path
//   4. Fall back to default site
func SiteContext(siteRepo repository.SiteRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        var siteID uint

        // 1. Explicit header (admin switching)
        if headerID := c.GetHeader("X-Site-ID"); headerID != "" {
            // parse and validate
        }

        // 2. Domain matching
        if siteID == 0 {
            host := c.Request.Host
            site, _ := siteRepo.FindByDomain(c.Request.Context(), host)
            if site != nil {
                siteID = site.ID
            }
        }

        // 3. Path prefix matching
        if siteID == 0 {
            // check path against registered base_paths
        }

        // 4. Default site
        if siteID == 0 {
            site, _ := siteRepo.FindDefault(c.Request.Context())
            if site != nil {
                siteID = site.ID
            }
        }

        c.Set(SiteContextKey, siteID)
        c.Next()
    }
}

// GetSiteID retrieves the current site ID from Gin context.
func GetSiteID(c *gin.Context) uint {
    val, exists := c.Get(SiteContextKey)
    if !exists {
        return 0
    }
    return val.(uint)
}
```

**Site repository:**

```go
// backend/internal/repository/site_repository.go
type SiteRepository interface {
    Create(ctx context.Context, site *model.Site) error
    FindByID(ctx context.Context, id uint) (*model.Site, error)
    FindBySlug(ctx context.Context, slug string) (*model.Site, error)
    FindByDomain(ctx context.Context, domain string) (*model.Site, error)
    FindDefault(ctx context.Context) (*model.Site, error)
    Update(ctx context.Context, site *model.Site) error
    Delete(ctx context.Context, id uint) error
    List(ctx context.Context) ([]*model.Site, error)
}
```

**Tasks:**
1. Write middleware tests (domain resolution, path resolution, fallback)
2. Implement SiteRepository interface and GORM impl
3. Implement SiteContext middleware
4. Register middleware globally in `cmd/server/main.go` (after CORS, before routes)
5. Run `cd backend && go test -v -race ./internal/middleware/...`

---

### 4.2.4 Site-level Config (0.5d)

**Modified files:**
- `backend/internal/handler/public/handler.go` — bootstrap response includes site config
- `backend/internal/handler/bootstrap/handler.go` — site-aware bootstrap
- `frontend/src/contexts/GlobalConfigContext.tsx` — read site config

**Site config JSON structure** (stored in `sites.config`):

```json
{
  "siteName": {"zh": "我的站点", "en": "My Site"},
  "siteDescription": {"zh": "...", "en": "..."},
  "logo": "/uploads/logo.png",
  "favicon": "/uploads/favicon.ico",
  "primaryColor": "#1a73e8",
  "googleAnalyticsId": "",
  "customCss": "",
  "customJs": "",
  "footer": {"zh": "...", "en": "..."},
  "seo": {
    "defaultTitle": {"zh": "...", "en": "..."},
    "titleTemplate": "%s - Site Name"
  }
}
```

**API:**

```
GET /admin/sites/:id/config      — get site config
PUT /admin/sites/:id/config      — update site config
```

**Tasks:**
1. Write handler tests
2. Implement config endpoints
3. Update bootstrap response to include site-specific config
4. Update frontend GlobalConfigContext to handle per-site config
5. Run full test suite

---

### 4.2.5 Site Management UI (1d)

**New files:**
- `frontend/src/pages/admin/sites/page.tsx` — Site list with create/edit/delete
- `frontend/src/pages/admin/sites/SiteEditor.tsx` — Site create/edit form
- `frontend/src/pages/admin/sites/SiteSwitcher.tsx` — Site switch dropdown (header component)
- `frontend/src/api/sites.ts`
- `frontend/src/pages/admin/sites/page.test.tsx`

**Backend handler:**
- `backend/internal/handler/site/handler.go`
- `backend/internal/handler/site/handler_test.go`

**API endpoints:**

```
GET    /admin/sites                — list all sites
GET    /admin/sites/:id            — get site details
POST   /admin/sites                — create site
PUT    /admin/sites/:id            — update site
DELETE /admin/sites/:id            — delete site (reject if default or has content)
PUT    /admin/sites/:id/default    — set as default site
```

**SiteSwitcher component** (in admin header):
- Dropdown showing all sites the user has access to
- Selected site stored in localStorage + React context
- All admin API calls include `X-Site-ID` header

**Tasks:**
1. Write handler tests (Go)
2. Implement site handler
3. Wire routes
4. Write frontend tests
5. Build site management pages
6. Build SiteSwitcher, integrate into AdminLayout
7. Run `pnpm lint && pnpm type-check && pnpm test`

---

### 4.2.6 Site-level Permissions (0.5d)

**Modified files:**
- `backend/internal/model/role.go` — UserRole already has `SiteID`
- `backend/internal/middleware/rbac.go` — Check site-scoped roles
- `backend/internal/handler/role/handler.go` — Site-scoped role assignment

**Logic:**
- A user can have different roles on different sites
- When `RequirePermission` runs, it loads roles matching the current site ID
- Global roles (SiteID=NULL in user_roles) apply to all sites
- Super admin bypasses all site checks

**Permission check priority:**
1. Global `super_admin` role -> allow everything
2. Site-specific role -> check permissions
3. Global role (non-super) -> check permissions
4. Deny

**Tasks:**
1. Write tests for site-scoped permission resolution
2. Update `FindByIDWithRoles` to accept optional siteID
3. Update `RequirePermission` to use site context
4. Run `cd backend && go test -v -race ./internal/middleware/...`

---

### 4.2.7 Site Data Export/Import (0.5d)

**Modified files:**
- `backend/internal/backup/service.go` — Add site-scoped export
- `backend/internal/backup/site_export.go` (new)
- `backend/internal/backup/site_export_test.go` (new)

**Export format (JSON archive):**

```json
{
  "version": "1.0",
  "exportedAt": "2026-03-11T...",
  "site": { /* site config */ },
  "articles": [ /* all articles for site */ ],
  "pages": [ /* all pages */ ],
  "categories": [],
  "tags": [],
  "media": [],  // metadata only; media files in separate tar
  "menus": [],
  "themes": [],
  "comments": []
}
```

**API:**

```
POST /admin/sites/:id/export      — trigger export, returns download URL
GET  /admin/sites/:id/export/:file — download export archive
POST /admin/sites/import           — upload and import site archive
```

**Tasks:**
1. Write export/import tests
2. Implement site-scoped export (JSON + media tar.gz)
3. Implement site import with conflict resolution (skip/overwrite)
4. Wire routes
5. Run `cd backend && go test -v -race ./internal/backup/...`

---

### 4.2.8 Subdomain vs Subpath Mode (0.5d)

**Modified files:**
- `backend/internal/middleware/site.go` — already handles both in SiteContext
- `backend/pkg/config/config.go` — Add `SITE_MODE` env var

**Configuration:**

```bash
# Env var: SITE_MODE=subdomain|subpath|domain
# subdomain: site1.example.com, site2.example.com
# subpath:   example.com/site1, example.com/site2
# domain:    site1.com, site2.com (each site has its own domain)
SITE_MODE=subdomain
```

**Subpath mode specifics:**
- Router strips site base_path prefix before matching routes
- All generated URLs include base_path prefix
- Static assets served from site-specific paths

**Tasks:**
1. Write tests for each mode
2. Implement path stripping for subpath mode
3. Implement subdomain extraction
4. Add SITE_MODE to config
5. Run `cd backend && go test -v -race ./internal/middleware/...`

---

## 4.3 Storage & Media Enhancement

**Current State**: `LocalStorage` implements `StorageProvider`. Media model stores URL, filename, MIME, size, width, height. No folders, no CDN, no chunked upload. The `S3 StorageProvider` plugin exists from Phase 3.

---

### 4.3.1 Storage Strategy Management (0.5d)

**New files:**
- `backend/internal/handler/storage/handler.go`
- `backend/internal/handler/storage/handler_test.go`
- `frontend/src/pages/admin/settings/storage/page.tsx`
- `frontend/src/pages/admin/settings/storage/page.test.tsx`

**System setting (stored in DB or site config):**

```json
{
  "storage": {
    "active": "local",           // or "s3", "oss", etc.
    "local": {
      "uploadDir": "./uploads"
    },
    "s3": {
      "bucket": "",
      "region": "",
      "accessKey": "",
      "secretKey": "",
      "endpoint": ""
    }
  }
}
```

**API:**

```
GET  /admin/settings/storage          — get current storage config
PUT  /admin/settings/storage          — update storage config, switch active provider
POST /admin/settings/storage/test     — test connection to configured provider
POST /admin/settings/storage/migrate  — migrate files between providers (async)
```

**Storage provider switching flow:**
1. Admin configures S3 credentials in UI
2. Clicks "Test Connection" -> backend tries `Save`/`Get`/`Delete` with test file
3. Admin clicks "Activate" -> active provider switches
4. Optionally clicks "Migrate existing files" -> background job copies local -> S3

**Tasks:**
1. Write handler tests
2. Implement storage settings handler
3. Implement provider test and migration logic
4. Build frontend settings page
5. Run full test suite

---

### 4.3.2 Media CDN Config (0.5d)

**Modified files:**
- `backend/internal/service/local_storage.go` — `URL()` method respects CDN prefix
- `backend/internal/handler/media/handler.go` — output URLs with CDN prefix
- `frontend/src/pages/admin/settings/storage/page.tsx` — CDN config section

**Implementation:**

```go
// Extend StorageProvider interface (backward compatible with default "")
// Or handle at handler/service level:

type CDNConfig struct {
    Enabled bool   `json:"enabled"`
    BaseURL string `json:"baseUrl"` // e.g. "https://cdn.example.com"
}

// In media handler, when returning media URLs:
func (h *Handler) resolveURL(path string) string {
    if h.cdnConfig.Enabled && h.cdnConfig.BaseURL != "" {
        return strings.TrimRight(h.cdnConfig.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
    }
    return "/uploads/" + path
}
```

**Tasks:**
1. Write URL resolution tests
2. Implement CDN config storage/retrieval
3. Update media handler to apply CDN prefix
4. Add CDN config UI section
5. Run tests

---

### 4.3.3 Image Processing Pipeline (0.5d)

**New files:**
- `backend/internal/service/image_processor.go`
- `backend/internal/service/image_processor_test.go`
- `backend/internal/service/task_queue.go`
- `backend/internal/service/task_queue_test.go`

**Image processing on upload:**

```go
// backend/internal/service/image_processor.go
type ImageProcessor struct {
    storage  provider.StorageProvider
    queue    *TaskQueue
    sizes    []ImageSize
}

type ImageSize struct {
    Name   string // "thumb", "medium", "large"
    Width  int
    Height int
    Fit    string // "cover", "contain", "fill"
}

var DefaultSizes = []ImageSize{
    {Name: "thumb", Width: 150, Height: 150, Fit: "cover"},
    {Name: "medium", Width: 800, Height: 0, Fit: "contain"},
    {Name: "large", Width: 1920, Height: 0, Fit: "contain"},
}

// ProcessImage queues async image processing tasks.
func (p *ImageProcessor) ProcessImage(ctx context.Context, mediaID uint, originalPath string) {
    p.queue.Enqueue(Task{
        Type:    "image_process",
        Payload: map[string]interface{}{"mediaID": mediaID, "path": originalPath},
    })
}
```

**Task queue** (simple in-process worker pool, not Redis — keep it lightweight):

```go
// backend/internal/service/task_queue.go
type TaskQueue struct {
    tasks   chan Task
    workers int
    handler func(Task)
    done    chan struct{}
}

func NewTaskQueue(workers int, handler func(Task)) *TaskQueue
func (q *TaskQueue) Start()
func (q *TaskQueue) Stop()
func (q *TaskQueue) Enqueue(task Task)
```

**Dependencies:** Use `golang.org/x/image` for basic resize, or `github.com/disintegration/imaging` for more features.

**Media model extension:**

```sql
-- +goose Up
ALTER TABLE media ADD COLUMN variants TEXT; -- JSON: {"thumb": "/uploads/img_thumb.webp", ...}
ALTER TABLE media ADD COLUMN processed BOOLEAN DEFAULT FALSE;
ALTER TABLE media ADD COLUMN folder_id INTEGER;
```

**Tasks:**
1. Write image processor tests (use test fixtures)
2. Write task queue tests
3. Implement ImageProcessor with WebP conversion and multi-size generation
4. Integrate with media upload handler
5. Run `cd backend && go test -v -race ./internal/service/...`

---

### 4.3.4 Media Folders (0.5d)

**New files:**
- `backend/internal/model/media_folder.go`
- `backend/internal/repository/media_folder_repository.go`
- `backend/internal/repository/media_folder_repository_impl.go`
- `frontend/src/pages/admin/media/FolderTree.tsx`

**Model:**

```go
type MediaFolder struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Name      string    `gorm:"not null;size:100" json:"name"`
    ParentID  *uint     `gorm:"index" json:"parentId"`
    SiteID    *uint     `gorm:"index" json:"siteId"`
    SortOrder int       `gorm:"default:0" json:"sortOrder"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
```

**API:**

```
GET    /admin/media/folders          — list folder tree
POST   /admin/media/folders          — create folder
PUT    /admin/media/folders/:id      — rename/move folder
DELETE /admin/media/folders/:id      — delete folder (reject if non-empty, or move to root)
PUT    /admin/media/:id/move         — move media to folder
```

**Frontend:**
- Folder tree sidebar in media library page
- Drag-and-drop media between folders
- Folder breadcrumb navigation

**Tasks:**
1. Write model and repository tests
2. Implement folder model and repository
3. Add folder endpoints to media handler
4. Build FolderTree frontend component
5. Integrate with existing media page
6. Run full test suite

---

### 4.3.5 Chunked Upload (0.5d)

**New files:**
- `backend/internal/handler/media/chunked.go`
- `backend/internal/handler/media/chunked_test.go`
- `frontend/src/components/feature/ChunkedUploader.tsx`
- `frontend/src/components/feature/ChunkedUploader.test.tsx`

**API (resumable upload protocol, simplified TUS-like):**

```
POST   /admin/media/upload/init     — initialize upload, returns uploadId
  Body: { filename, size, mimeType, chunkSize }
  Response: { uploadId, chunkSize, totalChunks }

PUT    /admin/media/upload/:uploadId/chunk/:index  — upload a chunk
  Body: binary chunk data
  Response: { received: true, index }

POST   /admin/media/upload/:uploadId/complete      — finalize, merge chunks
  Response: { media: { id, url, ... } }

GET    /admin/media/upload/:uploadId/status         — get upload progress
  Response: { uploadId, receivedChunks: [0,1,3], totalChunks: 5 }

DELETE /admin/media/upload/:uploadId                — cancel upload, clean temp
```

**Backend implementation:**

```go
// Temp directory: {uploadDir}/.chunks/{uploadId}/
// Each chunk saved as: chunk_{index}
// On complete: merge chunks sequentially, run through image pipeline, clean up

type ChunkedUploadState struct {
    UploadID   string   `json:"uploadId"`
    Filename   string   `json:"filename"`
    Size       int64    `json:"size"`
    MimeType   string   `json:"mimeType"`
    ChunkSize  int      `json:"chunkSize"`
    TotalChunks int     `json:"totalChunks"`
    Received   []int    `json:"received"` // indices of received chunks
    CreatedAt  time.Time `json:"createdAt"`
}
```

**Frontend ChunkedUploader:**
- Default chunk size: 2MB
- Parallel chunk uploads (3 concurrent)
- Progress bar per file
- Resume support: check status, re-upload missing chunks
- Falls back to regular upload for files < 5MB

**Tasks:**
1. Write chunked upload handler tests
2. Implement chunk init/upload/complete/status/cancel
3. Write frontend component tests
4. Build ChunkedUploader component
5. Integrate with media upload page (replace for large files)
6. Run full test suite

---

## 4.4 Data Migration

**Current State**: Basic article import/export (Markdown) exists in `backend/internal/handler/article/handler.go` (AdminImportMarkdown/AdminExportMarkdown). No migration framework for other platforms.

---

### 4.4.1 Migration Framework Design (0.5d)

**New files:**
- `backend/internal/migration/provider.go` — interface
- `backend/internal/migration/runner.go` — orchestration
- `backend/internal/migration/runner_test.go`

**Interface:**

```go
// backend/internal/migration/provider.go
package migration

import "context"

// MigrationProvider defines the contract for importing data from external platforms.
type MigrationProvider interface {
    // ID returns the provider identifier (e.g. "wordpress", "halo", "markdown").
    ID() string

    // Name returns a human-readable name.
    Name() string

    // Validate checks if the import source is valid (file format, required fields).
    Validate(ctx context.Context, source *ImportSource) (*ValidationResult, error)

    // Import runs the migration. Calls progressFn periodically with updates.
    Import(ctx context.Context, source *ImportSource, opts ImportOptions, progressFn ProgressFunc) (*ImportResult, error)
}

type ImportSource struct {
    FilePath    string            // uploaded file path
    Format      string            // "wxr", "json", "markdown", "zip"
    Metadata    map[string]string // extra config from user
}

type ImportOptions struct {
    SiteID         uint
    DefaultAuthor  string
    DefaultLocale  string
    DuplicateMode  string // "skip" | "overwrite" | "rename"
    ImportMedia    bool   // download and re-upload referenced media
}

type ProgressFunc func(progress Progress)

type Progress struct {
    Phase       string  // "parsing", "importing_articles", "importing_media", etc.
    Current     int
    Total       int
    Percentage  float64
    Message     string
    Errors      []string
}

type ValidationResult struct {
    Valid       bool
    Articles    int
    Pages       int
    Categories  int
    Tags        int
    Media       int
    Warnings    []string
    Errors      []string
}

type ImportResult struct {
    ArticlesImported  int
    PagesImported     int
    CategoriesCreated int
    TagsCreated       int
    MediaDownloaded   int
    Skipped           int
    Errors            []ImportError
    Duration          time.Duration
}

type ImportError struct {
    Resource string
    Title    string
    Error    string
}
```

**Migration runner (manages async import jobs):**

```go
// backend/internal/migration/runner.go
type Runner struct {
    providers map[string]MigrationProvider
    jobs      map[string]*MigrationJob  // jobID -> job
    mu        sync.RWMutex
    db        *gorm.DB
}

type MigrationJob struct {
    ID        string
    Provider  string
    Status    string  // "pending", "running", "completed", "failed"
    Progress  Progress
    Result    *ImportResult
    StartedAt time.Time
    Error     error
}

func (r *Runner) RegisterProvider(p MigrationProvider)
func (r *Runner) StartImport(ctx context.Context, providerID string, source *ImportSource, opts ImportOptions) (string, error)
func (r *Runner) GetJob(jobID string) *MigrationJob
func (r *Runner) ListJobs() []*MigrationJob
```

**Tasks:**
1. Write runner tests (mock provider)
2. Implement Runner with job management
3. Register in provider.Registry
4. Run `cd backend && go test -v -race ./internal/migration/...`

---

### 4.4.2 WordPress WXR Import (0.5d)

**New files:**
- `backend/internal/migration/wordpress.go`
- `backend/internal/migration/wordpress_test.go`
- `backend/internal/migration/testdata/wordpress-sample.xml`

**Implementation:**

```go
// backend/internal/migration/wordpress.go
type WordPressMigration struct {
    db         *gorm.DB
    storage    provider.StorageProvider
    httpClient *http.Client
}

func (w *WordPressMigration) ID() string   { return "wordpress" }
func (w *WordPressMigration) Name() string { return "WordPress (WXR)" }
```

**WXR parsing** (WordPress eXtended RSS):
- Parse `<item>` elements with `<wp:post_type>` = "post" or "page"
- Map `<category domain="category">` to Category model
- Map `<category domain="post_tag">` to Tag model
- Convert HTML body to clean HTML (sanitize)
- Download `<wp:attachment_url>` images and re-upload via StorageProvider
- Map `<wp:status>` (publish/draft/private) to ArticleStatus
- Parse `<wp:postmeta>` for SEO fields if Yoast/RankMath metadata present

**Test with fixture:**

```go
func TestWordPressMigration_Import(t *testing.T) {
    // Use testdata/wordpress-sample.xml
    // Assert correct article/page/category/tag counts
    // Assert media download attempted
}
```

**Tasks:**
1. Create sample WXR fixture
2. Write import tests
3. Implement WXR XML parser
4. Implement media download logic
5. Register provider
6. Run `cd backend && go test -v -race ./internal/migration/...`

---

### 4.4.3 Halo Import (0.5d)

**New files:**
- `backend/internal/migration/halo.go`
- `backend/internal/migration/halo_test.go`
- `backend/internal/migration/testdata/halo-sample.json`

**Halo export format** (JSON):
- Posts with Markdown content
- Categories and tags
- Attachments with URLs

**Implementation similar to WordPress but simpler JSON parsing.**

**Tasks:**
1. Create Halo sample fixture
2. Write import tests
3. Implement Halo JSON parser
4. Register provider
5. Run tests

---

### 4.4.4 Markdown Batch Import (0.5d)

**New files:**
- `backend/internal/migration/markdown_batch.go`
- `backend/internal/migration/markdown_batch_test.go`

**Input:** ZIP file containing a directory tree of `.md` files with YAML front-matter.

**Front-matter mapping:**

```yaml
---
title: "My Post"
date: 2025-01-15
tags: [go, web]
categories: [programming]
slug: my-post
draft: false
description: "SEO description"
cover: images/cover.jpg
---

Post content here...
```

**Directory structure handling:**
- Top-level directories become categories
- Files at root get no category
- `images/` subdirectory: upload referenced images

**Tasks:**
1. Create test fixtures (ZIP with sample markdown files)
2. Write import tests
3. Implement front-matter parser and batch importer
4. Register provider
5. Run tests

---

### 4.4.5 Migration Progress & Logging (0.5d)

**New/modified files:**
- `backend/internal/handler/migration/handler.go`
- `backend/internal/handler/migration/handler_test.go`
- `frontend/src/pages/admin/settings/migration/page.tsx`
- `frontend/src/pages/admin/settings/migration/page.test.tsx`

**API:**

```
GET    /admin/migration/providers          — list available migration providers
POST   /admin/migration/validate           — validate uploaded file
  Body: multipart/form-data { file, provider }
  Response: { valid, articles, pages, ... }
POST   /admin/migration/import             — start async import
  Body: multipart/form-data { file, provider, options }
  Response: { jobId }
GET    /admin/migration/jobs                — list migration jobs
GET    /admin/migration/jobs/:id            — get job status + progress
GET    /admin/migration/jobs/:id/log        — get detailed error log
DELETE /admin/migration/jobs/:id            — cancel running job
```

**Frontend migration page:**
1. Select migration provider from dropdown
2. Upload file (drag-and-drop zone)
3. Click "Validate" -> show summary (N articles, M categories, etc.)
4. Configure options (duplicate handling, default author, import media?)
5. Click "Start Import" -> show live progress bar
6. On complete: show result summary with error count link

**Progress via polling** (SSE/WebSocket overkill for this):
- Frontend polls `GET /admin/migration/jobs/:id` every 2 seconds during import
- Shows progress bar, current phase, item counts

**Tasks:**
1. Write handler tests
2. Implement migration handler (file upload, job management)
3. Write frontend tests
4. Build migration settings page
5. Wire routes in `cmd/server/main.go`
6. Run full test suite

---

## 4.5 Operations & Monitoring

**Current State**: `/health` endpoint checks DB connectivity. `/metrics` returns publish/validation/rollback/public_get counters. Backup is manual trigger via `/admin/backups/trigger`. Audit log records action/actor/resource/result/details.

---

### 4.5.1 System Status Dashboard (0.5d)

**New files:**
- `backend/internal/handler/system/handler.go`
- `backend/internal/handler/system/handler_test.go`
- `frontend/src/pages/admin/system/page.tsx`
- `frontend/src/pages/admin/system/page.test.tsx`

**API:**

```
GET /admin/system/status     — comprehensive system status (admin only)
```

**Response:**

```json
{
  "runtime": {
    "goVersion": "go1.25.0",
    "os": "linux",
    "arch": "amd64",
    "cpuCount": 4,
    "goroutines": 42,
    "uptime": 86400
  },
  "memory": {
    "allocMB": 45.2,
    "totalAllocMB": 1200.5,
    "sysMB": 72.0,
    "gcPauseMs": 0.5
  },
  "database": {
    "type": "sqlite",
    "version": "3.46.0",
    "openConnections": 1,
    "maxOpenConnections": 1,
    "sizeBytes": 5242880,
    "tableCount": 18
  },
  "storage": {
    "provider": "local",
    "uploadDirSizeMB": 256.0,
    "diskFreeMB": 10240.0,
    "mediaCount": 142
  },
  "content": {
    "articles": 45,
    "pages": 12,
    "comments": 230,
    "media": 142,
    "users": 3,
    "sites": 1
  },
  "requestStats": {
    "last24h": {
      "total": 15000,
      "errors": 23,
      "avgLatencyMs": 12.5
    }
  },
  "plugins": [],
  "backup": {
    "lastBackupAt": "2026-03-10T...",
    "backupCount": 8,
    "nextScheduledAt": "2026-03-11T..."
  }
}
```

**Implementation:**

```go
// backend/internal/handler/system/handler.go
func (h *Handler) GetStatus(c *gin.Context) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    // DB stats
    sqlDB, _ := h.db.DB()
    dbStats := sqlDB.Stats()

    // Disk usage (upload dir)
    diskFree := getDiskFree(h.uploadDir)

    // Content counts
    var articleCount, pageCount, mediaCount int64
    h.db.Model(&model.Article{}).Count(&articleCount)
    // ...

    c.JSON(200, statusResponse{...})
}
```

**Frontend dashboard page:**
- Summary cards: CPU usage, memory, disk, DB size
- Content count cards: articles, pages, comments, users
- Recent request chart (simple sparkline from `/metrics`)
- Backup status card with "last backup" time
- Plugin status list

**Tasks:**
1. Write handler tests
2. Implement system status handler
3. Write frontend tests
4. Build system dashboard page
5. Wire route (admin only, require `system:read` permission)
6. Run full test suite

---

### 4.5.2 Auto-backup Scheduling (0.5d)

**Modified files:**
- `backend/internal/backup/service.go` — add scheduler
- `backend/internal/backup/scheduler.go` (new)
- `backend/internal/backup/scheduler_test.go` (new)
- `backend/pkg/config/config.go` — add backup config env vars

**Configuration:**

```bash
# Backup schedule (cron expression)
BACKUP_SCHEDULE=0 2 * * *     # daily at 2 AM
BACKUP_MAX_KEEP=10             # keep last 10 backups (already exists)
BACKUP_ENABLED=true
```

**Implementation:**

```go
// backend/internal/backup/scheduler.go
package backup

import (
    "context"
    "log/slog"
    "time"

    "github.com/robfig/cron/v3"
)

type Scheduler struct {
    cron      *cron.Cron
    service   *Service
    logger    *slog.Logger
    schedule  string
    enabled   bool
    nextRunAt time.Time
}

func NewScheduler(svc *Service, schedule string, enabled bool) *Scheduler {
    return &Scheduler{
        cron:     cron.New(cron.WithSeconds()),
        service:  svc,
        logger:   slog.Default(),
        schedule: schedule,
        enabled:  enabled,
    }
}

func (s *Scheduler) Start() error {
    if !s.enabled {
        s.logger.Info("Backup scheduler disabled")
        return nil
    }

    _, err := s.cron.AddFunc(s.schedule, func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
        defer cancel()

        s.logger.Info("Starting scheduled backup")
        record, err := s.service.RunBackup(ctx)
        if err != nil {
            s.logger.Error("Scheduled backup failed", "error", err)
            return
        }
        s.logger.Info("Scheduled backup completed", "filename", record.Filename, "size", record.Size)
    })
    if err != nil {
        return fmt.Errorf("invalid cron schedule %q: %w", s.schedule, err)
    }

    s.cron.Start()
    s.logger.Info("Backup scheduler started", "schedule", s.schedule)
    return nil
}

func (s *Scheduler) Stop() {
    s.cron.Stop()
}

func (s *Scheduler) NextRun() time.Time {
    entries := s.cron.Entries()
    if len(entries) > 0 {
        return entries[0].Next
    }
    return time.Time{}
}
```

**Wire into `cmd/server/main.go`:**

```go
// After backup service initialization:
backupScheduler := backup.NewScheduler(backupSvc, cfg.BackupSchedule, cfg.BackupEnabled)
if err := backupScheduler.Start(); err != nil {
    log.Error("Failed to start backup scheduler", "error", err)
}
// In shutdown:
backupScheduler.Stop()
```

**API extension:**

```
GET  /admin/backups/schedule     — get schedule config + next run time
PUT  /admin/backups/schedule     — update schedule (runtime, persisted to DB/config)
```

**Tasks:**
1. Add `github.com/robfig/cron/v3` dependency: `cd backend && go get github.com/robfig/cron/v3`
2. Write scheduler tests (mock backup service)
3. Implement Scheduler
4. Add config env vars
5. Wire into main.go
6. Add schedule API endpoints
7. Run `cd backend && go test -v -race ./internal/backup/...`

---

### 4.5.3 Remote Backup Storage (0.5d)

**New files:**
- `backend/internal/backup/remote.go`
- `backend/internal/backup/remote_test.go`

**Implementation:** After a backup completes, optionally upload to remote storage.

```go
// backend/internal/backup/remote.go
type RemoteBackupConfig struct {
    Enabled   bool   `json:"enabled"`
    Provider  string `json:"provider"`  // "s3", "oss", etc.
    Bucket    string `json:"bucket"`
    Prefix    string `json:"prefix"`    // e.g. "backups/site-1/"
    // Credentials via StorageProvider
}

// UploadToRemote uploads a backup file to the configured remote storage.
func (s *Service) UploadToRemote(ctx context.Context, filename string, remote provider.StorageProvider, prefix string) error {
    localPath := filepath.Join(s.backupDir, filename)
    f, err := os.Open(localPath)
    if err != nil {
        return fmt.Errorf("open backup file: %w", err)
    }
    defer f.Close()

    info, _ := f.Stat()
    remotePath := prefix + filename
    _, err = remote.Save(ctx, remotePath, f, info.Size())
    return err
}
```

**Integration with scheduler:**

```go
// In scheduler's backup function:
if s.remoteConfig.Enabled {
    remoteProvider := registry.Get(s.remoteConfig.Provider).(provider.StorageProvider)
    if err := s.service.UploadToRemote(ctx, record.Filename, remoteProvider, s.remoteConfig.Prefix); err != nil {
        s.logger.Error("Remote backup upload failed", "error", err)
        // Don't fail the backup — local copy exists
    }
}
```

**Backup record extension:**

```sql
ALTER TABLE backup_records ADD COLUMN remote_path VARCHAR(500);
ALTER TABLE backup_records ADD COLUMN remote_provider VARCHAR(50);
```

**Tasks:**
1. Write remote upload tests (mock StorageProvider)
2. Implement UploadToRemote
3. Integrate with scheduler
4. Add remote config API
5. Run tests

---

### 4.5.4 Audit Log Enhancement (0.5d)

**Modified files:**
- `backend/internal/model/audit_event.go`
- `backend/internal/handler/auditlog/handler.go`
- `frontend/src/pages/admin/audit-logs/page.tsx`

**Model extension:**

```sql
-- +goose Up
ALTER TABLE audit_events ADD COLUMN ip_address VARCHAR(45);
ALTER TABLE audit_events ADD COLUMN user_agent TEXT;
ALTER TABLE audit_events ADD COLUMN severity VARCHAR(10) DEFAULT 'info';
  -- severity: "info", "warning", "critical"
ALTER TABLE audit_events ADD COLUMN site_id INTEGER;
```

**Updated AuditEvent model:**

```go
type AuditEvent struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Action    string    `gorm:"not null;size:50;index" json:"action"`
    Actor     string    `gorm:"not null;size:100;index" json:"actor"`
    Resource  string    `gorm:"not null;size:100" json:"resource"`
    Result    string    `gorm:"not null;size:20" json:"result"`
    Details   string    `gorm:"type:text" json:"details"`
    IPAddress string    `gorm:"size:45" json:"ipAddress"`
    UserAgent string    `gorm:"type:text" json:"userAgent"`
    Severity  string    `gorm:"size:10;default:'info'" json:"severity"`
    SiteID    *uint     `gorm:"index" json:"siteId"`
    CreatedAt time.Time `gorm:"autoCreateTime;index" json:"createdAt"`
}
```

**Sensitive operation detection:**

```go
var SensitiveActions = map[string]string{
    "user.delete":       "critical",
    "user.role_change":  "warning",
    "backup.restore":    "critical",
    "site.delete":       "critical",
    "settings.update":   "warning",
    "plugin.install":    "warning",
    "plugin.uninstall":  "warning",
}
```

**Audit log middleware** (capture IP and User-Agent automatically):

```go
// backend/internal/middleware/audit.go
func AuditContext() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Set("client_ip", c.ClientIP())
        c.Set("user_agent", c.GetHeader("User-Agent"))
        c.Next()
    }
}
```

**Frontend enhancement:**
- Add severity badge (color-coded) to audit log table
- Add IP address column
- Add filter by severity
- Highlight critical events with red background

**Tasks:**
1. Create migration for new columns
2. Update AuditEvent model
3. Implement AuditContext middleware
4. Update audit log writer to capture IP/UA/severity
5. Update frontend audit log page
6. Run full test suite

---

### 4.5.5 Health Check Enhancement (0.5d)

**Modified files:**
- `backend/cmd/server/main.go` — expand `/health` endpoint

**Enhanced health response:**

```json
{
  "status": "healthy",
  "timestamp": "2026-03-11T...",
  "version": "1.2.0",
  "checks": {
    "database": {
      "status": "healthy",
      "latencyMs": 2,
      "type": "sqlite",
      "version": "3.46.0"
    },
    "storage": {
      "status": "healthy",
      "provider": "local",
      "writable": true
    },
    "scheduler": {
      "status": "healthy",
      "nextBackupAt": "2026-03-11T02:00:00Z",
      "schedulerRunning": true
    },
    "plugins": {
      "status": "healthy",
      "active": 3,
      "errored": 0
    },
    "memory": {
      "status": "healthy",
      "usedMB": 45.2,
      "thresholdMB": 512
    }
  }
}
```

**Implementation:**

```go
// backend/internal/health/checker.go
type HealthChecker struct {
    checks []NamedCheck
}

type NamedCheck struct {
    Name  string
    Check func(ctx context.Context) CheckResult
}

type CheckResult struct {
    Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
    Details   map[string]interface{} `json:"details,omitempty"`
}

func (h *HealthChecker) RunAll(ctx context.Context) map[string]CheckResult {
    results := make(map[string]CheckResult)
    for _, nc := range h.checks {
        results[nc.Name] = nc.Check(ctx)
    }
    return results
}
```

**Checks to implement:**
1. **Database**: Ping + query version + measure latency
2. **Storage**: Write a temp file, read it, delete it
3. **Scheduler**: Check if backup scheduler goroutine is alive
4. **Memory**: Check if Go heap exceeds threshold
5. **Plugins** (Phase 3): Check each plugin's health status

**HTTP status code logic:**
- All healthy -> 200
- Any degraded -> 200 (with degraded status in body)
- Any unhealthy -> 503

**Tasks:**
1. Write health checker tests
2. Implement HealthChecker with individual check functions
3. Replace inline `/health` handler with HealthChecker
4. Add `/health/live` (simple liveness: always 200) and `/health/ready` (full check)
5. Run `cd backend && go test -v -race ./internal/health/...`

---

## Verification Commands

Run these commands at each milestone to ensure nothing is broken:

```bash
# Backend: compile + vet + test
cd backend && go build -o /dev/null ./cmd/server/ && go vet ./... && go test -v -race ./...

# Frontend: lint + type-check + test
pnpm lint && pnpm type-check && pnpm test

# Full quality gate (mirrors CI)
make check && cd backend && go test -v -race ./...

# Integration: start backend and test key endpoints
cd backend && PORT=8088 DB_DSN="file:./data/test.db?cache=shared&mode=rwc" \
  JWT_SECRET=test JWT_REFRESH_SECRET=test ENV=test \
  go run ./cmd/server/ &
sleep 2
curl -s --noproxy '*' http://127.0.0.1:8088/health | jq .
curl -s --noproxy '*' http://127.0.0.1:8088/version | jq .
kill %1
```

---

## Migration Safety Checklist

Before deploying each sub-phase:

- [ ] All goose migrations have matching `-- +goose Down` blocks
- [ ] New columns are nullable or have defaults (no breaking existing rows)
- [ ] `site_id` backfill migration runs in a transaction
- [ ] Existing API responses unchanged (new fields only additive)
- [ ] Legacy middleware (`RequireAdmin`, etc.) still works alongside new RBAC
- [ ] JWT token format backward compatible (existing tokens still valid)
- [ ] No GORM AutoMigrate for schema changes (goose only)
- [ ] New indexes don't lock tables on large datasets (use `CREATE INDEX CONCURRENTLY` for PostgreSQL)

---

## Dependency Graph

```
4.1.1 RBAC Model
  └─> 4.1.2 Built-in Roles (depends on 4.1.1)
  └─> 4.1.3 Middleware Refactor (depends on 4.1.1)
       └─> 4.1.4 Permission UI (depends on 4.1.3)
  └─> 4.1.5 Registration (depends on 4.1.1)
  └─> 4.1.6 OAuth2 Interface (independent, interface only)
  └─> 4.1.7 Profile Page (depends on 4.1.1)

4.2.1 Site Model
  └─> 4.2.2 Data Isolation (depends on 4.2.1)
       └─> 4.2.3 Site Middleware (depends on 4.2.2)
       └─> 4.2.4 Site Config (depends on 4.2.2)
       └─> 4.2.5 Site UI (depends on 4.2.3)
       └─> 4.2.6 Site Permissions (depends on 4.1.3 + 4.2.3)
       └─> 4.2.7 Site Export/Import (depends on 4.2.2)
       └─> 4.2.8 Subdomain/Subpath (depends on 4.2.3)

4.3.1-4.3.5 are mostly independent of each other, but:
  4.3.3 Image Pipeline depends on 4.3.1 (storage strategy)
  4.3.4 Media Folders is independent
  4.3.5 Chunked Upload is independent

4.4.1 Migration Framework
  └─> 4.4.2 WordPress (depends on 4.4.1)
  └─> 4.4.3 Halo (depends on 4.4.1)
  └─> 4.4.4 Markdown Batch (depends on 4.4.1)
  └─> 4.4.5 Progress UI (depends on 4.4.1)

4.5.1-4.5.5 are mostly independent of each other
```

---

## Recommended Implementation Order

**Week 1: RBAC Foundation**
1. 4.1.1 RBAC Model Design
2. 4.1.2 Built-in Roles
3. 4.1.3 Permission Middleware Refactor
4. 4.1.6 OAuth2 Interface (quick, interface only)

**Week 2: Users + Site Foundation**
5. 4.1.4 Permission Management UI
6. 4.1.5 User Registration & Invite
7. 4.1.7 Profile Page
8. 4.2.1 Site Model Design

**Week 3: Multi-site + Storage**
9. 4.2.2 Data Isolation with site_id
10. 4.2.3 Site Context Middleware
11. 4.2.4 Site-level Config
12. 4.2.5 Site Management UI
13. 4.3.1 Storage Strategy Management
14. 4.3.2 Media CDN Config

**Week 4: Media + Migration + Ops**
15. 4.2.6 Site-level Permissions
16. 4.2.7 Site Data Export/Import
17. 4.2.8 Subdomain vs Subpath
18. 4.3.3 Image Processing Pipeline
19. 4.3.4 Media Folders
20. 4.3.5 Chunked Upload
21. 4.4.1 Migration Framework
22. 4.4.2 WordPress WXR Import
23. 4.4.3 Halo Import
24. 4.4.4 Markdown Batch Import
25. 4.4.5 Migration Progress & Logging
26. 4.5.1 System Status Dashboard
27. 4.5.2 Auto-backup Scheduling
28. 4.5.3 Remote Backup Storage
29. 4.5.4 Audit Log Enhancement
30. 4.5.5 Health Check Enhancement
