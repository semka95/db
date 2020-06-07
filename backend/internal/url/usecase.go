package url

import (
	"context"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
)

// Usecase represent the url's usecases
type Usecase interface {
	GetByID(ctx context.Context, id string) (*models.URL, error)
	Update(ctx context.Context, updateURL *models.UpdateURL, user auth.Claims) error
	Store(ctx context.Context, createURL *models.CreateURL) (*models.URL, error)
	Delete(ctx context.Context, id string) error
}
