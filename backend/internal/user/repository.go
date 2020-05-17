package user

import (
	"context"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Repository represent the User's repository contract
type Repository interface {
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Create(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
