package user

import (
	"context"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
)

// Repository represent the User's repository contract
type Repository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, url *models.User) error
	Create(ctx context.Context, u *models.User) error
	Delete(ctx context.Context, id string) error
}
