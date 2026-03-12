-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    site_id     INTEGER,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    resource    VARCHAR(50) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    description TEXT,
    UNIQUE(resource, action)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    rbac_role_id  INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (rbac_role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id   INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    site_id   INTEGER,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_roles_site ON roles(site_id);

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
