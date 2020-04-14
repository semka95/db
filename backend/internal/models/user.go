package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents the User model
type User struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	FullName  string             `json:"full_name,omitempty" bson:"full_name,omitempty"`
	Email     string             `json:"email,omitempty" bson:"email,omitempty"`
	Password  string             `json:"password,omitempty" bson:"password,omitempty"`
	CreatedAt time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

// NewUser creates instance of User model
func NewUser() *User {
	id, _ := primitive.ObjectIDFromHex("507f191e810c19729de860ea")
	return &User{
		ID:        id,
		FullName:  "John Doe",
		Email:     "test@example.com",
		Password:  "",
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
	}
}
