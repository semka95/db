package url

import (
	"context"

	"github.com/bxcodec/go-clean-arch/internal/models"
)

// Repository represent the URL's repository contract
type Repository interface {
	GetByID(ctx context.Context, id string) (*models.URL, error)
	Update(ctx context.Context, url *models.URL) error
	Store(ctx context.Context, u *models.URL) error
	Delete(ctx context.Context, id string) error
}
