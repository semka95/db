package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/semka95/shortener/backend/web/auth"
)

// User represents the User model
type User struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	FullName       string             `json:"full_name" bson:"full_name"`
	Email          string             `json:"email" bson:"email"`
	HashedPassword string             `json:"-" bson:"hashed_password"`
	Roles          []string           `json:"roles" bson:"roles"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

// CreateUser represents data to create new User
type CreateUser struct {
	FullName string `json:"full_name" validate:"omitempty,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=30"`
}

// UpdateUser represents data to update User
type UpdateUser struct {
	ID              primitive.ObjectID `json:"id" validate:"required"`
	FullName        *string            `json:"full_name" validate:"omitempty,max=30"`
	Email           *string            `json:"email" validate:"omitempty,email"`
	CurrentPassword string             `json:"current_password" validate:"required,min=8,max=30"`
	NewPassword     *string            `json:"new_password" validate:"omitempty,min=8,max=30"`
}

// UserUsecase represents the User's usecases
type UserUsecase interface {
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user UpdateUser, claims *auth.Claims) error
	Create(ctx context.Context, user CreateUser) (*User, error)
	Delete(ctx context.Context, id string) error
	Authenticate(ctx context.Context, now time.Time, email, password string) (*auth.Claims, error)
}

// UserRepository represents the User's repository contract
type UserRepository interface {
	GetByID(ctx context.Context, id primitive.ObjectID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Create(ctx context.Context, user *User) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
