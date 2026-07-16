package model

import (
	"slices"
	"testing"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		role  Role
		valid bool
	}{
		{"admin is valid", RoleAdmin, true},
		{"editor is valid", RoleEditor, true},
		{"invalid role", Role("invalid"), false},
		{"empty role", Role(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.valid {
				t.Errorf("Role.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestRole_String(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want string
	}{
		{"admin string", RoleAdmin, "admin"},
		{"editor string", RoleEditor, "editor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("Role.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	if got := user.TableName(); got != "users" {
		t.Errorf("User.TableName() = %v, want %v", got, "users")
	}
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			user: User{
				Username:     "testuser",
				PasswordHash: "hashedpassword",
				Role:         RoleAdmin,
			},
			wantErr: false,
		},
		{
			name: "missing username",
			user: User{
				PasswordHash: "hashedpassword",
				Role:         RoleAdmin,
			},
			wantErr: true,
			errMsg:  "username is required",
		},
		{
			name: "missing password hash",
			user: User{
				Username: "testuser",
				Role:     RoleAdmin,
			},
			wantErr: true,
			errMsg:  "password hash is required",
		},
		{
			name: "invalid role",
			user: User{
				Username:     "testuser",
				PasswordHash: "hashedpassword",
				Role:         Role("invalid"),
			},
			wantErr: true,
			errMsg:  "role must be 'admin' or 'editor'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("User.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestUser_EffectivePermissionKeys(t *testing.T) {
	t.Run("super admin uses wildcard", func(t *testing.T) {
		user := User{IsSuperAdmin: true}
		if got := user.EffectivePermissionKeys(); !slices.Equal(got, []string{"*:*"}) {
			t.Fatalf("EffectivePermissionKeys() = %v, want wildcard", got)
		}
	})

	t.Run("loaded RBAC roles are flattened and deduplicated", func(t *testing.T) {
		read := Permission{Resource: "pages", Action: "read"}
		update := Permission{Resource: "pages", Action: "update"}
		user := User{
			UserRoles: []UserRole{
				{Role: RBACRole{Permissions: []Permission{update, read}}},
				{Role: RBACRole{Permissions: []Permission{read}}},
			},
		}

		want := []string{"pages:read", "pages:update"}
		if got := user.EffectivePermissionKeys(); !slices.Equal(got, want) {
			t.Fatalf("EffectivePermissionKeys() = %v, want %v", got, want)
		}
	})

	t.Run("legacy editor cannot publish content", func(t *testing.T) {
		user := User{Role: RoleEditor}
		if user.HasRBACPermission("pages", "publish") {
			t.Fatal("legacy editor unexpectedly has pages:publish")
		}
		if user.HasRBACPermission("articles", "publish") {
			t.Fatal("legacy editor unexpectedly has articles:publish")
		}
		if !user.HasRBACPermission("pages", "update") {
			t.Fatal("legacy editor should retain pages:update")
		}
	})
}
