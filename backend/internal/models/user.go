package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents the User model
type User struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	FullName       string             `json:"full_name" bson:"full_name"`
	Email          string             `json:"email" bson:"email"`
	HashedPassword string             `json:"-" bson:"hashed_password"`
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
	ID       primitive.ObjectID `json:"id" validate:"required"`
	FullName *string            `json:"full_name" validate:"omitempty,max=30"`
	Email    *string            `json:"email" validate:"omitempty,email"`
	Password *string            `json:"password" validate:"omitempty,min=8,max=30"`
}
