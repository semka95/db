package user

import (
	"context"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
)

// Usecase represent the user's usecases
type Usecase interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, user *models.UpdateUser) error
	Create(ctx context.Context, user *models.CreateUser) (*models.User, error)
	Delete(ctx context.Context, id string) error
	Authenticate(ctx context.Context, now time.Time, email, password string) (*auth.Claims, error)
}
