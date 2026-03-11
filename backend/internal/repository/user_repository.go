package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *model.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id uint) (*model.User, error)

	// FindByUsername finds a user by username
	FindByUsername(ctx context.Context, username string) (*model.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *model.User) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id uint) error

	// List returns a paginated list of users
	List(ctx context.Context, offset, limit int) ([]*model.User, int64, error)

	// CountSuperAdmins returns the number of super admin users
	CountSuperAdmins(ctx context.Context) (int64, error)
}
