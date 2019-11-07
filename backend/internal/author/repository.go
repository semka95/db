package author

import (
	"context"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
)

// Repository represent the author's repository contract
type Repository interface {
	GetByID(ctx context.Context, id int64) (*models.Author, error)
}
