package user

import (
	"context"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
)

// Usecase represent the user's usecases
type Usecase interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, user *models.UpdateUser) error
	Create(ctx context.Context, user *models.CreateUser) (*models.User, error)
	Delete(ctx context.Context, id string) error
}
