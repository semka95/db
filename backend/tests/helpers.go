package tests

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/web/auth"
)

// StringPointer returns pointer of a string
func StringPointer(s string) *string {
	return &s
}

// DatePointer returns pointer of a time.Time
func DatePointer(t time.Time) *time.Time {
	return &t
}

// NewUser creates instance of User model
func NewUser() *domain.User {
	id, _ := primitive.ObjectIDFromHex("507f191e810c19729de860ea")
	return &domain.User{
		ID:             id,
		FullName:       "John Doe",
		Email:          "test@example.com",
		HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS", // password
		Roles:          []string{auth.RoleUser},
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}
}

// NewUpdateUser creates instance of UpdateUser model
func NewUpdateUser() domain.UpdateUser {
	id, _ := primitive.ObjectIDFromHex("507f191e810c19729de860ea")
	return domain.UpdateUser{
		ID:              id,
		FullName:        StringPointer("John Doe"),
		Email:           StringPointer("test@example.com"),
		CurrentPassword: "password",
		NewPassword:     StringPointer("newpassword"),
	}
}

// NewCreateUser creates instance of CreateUser model
func NewCreateUser() domain.CreateUser {
	return domain.CreateUser{
		FullName: "John Doe",
		Email:    "test@example.com",
		Password: "newpassword",
	}
}

// NewUserBsonD creates instance of User model in bson format
func NewUserBsonD() bson.D {
	id, _ := primitive.ObjectIDFromHex("507f191e810c19729de860ea")
	return bson.D{
		{Key: "_id", Value: id},
		{Key: "full_name", Value: "John Doe"},
		{Key: "email", Value: "test@example.com"},
		{Key: "hashed_password", Value: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS"},
		{Key: "roles", Value: []string{auth.RoleUser}},
		{Key: "created_at", Value: time.Now().Truncate(time.Millisecond).UTC()},
		{Key: "updated_at", Value: time.Now().Truncate(time.Millisecond).UTC()},
	}
}

// NewURL creates instance of URL model
func NewURL() *domain.URL {
	return &domain.URL{
		ID:             "test123",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC(),
		UserID:         "507f191e810c19729de860ea",
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}
}

// NewCreateURL creates instance of CreateURL model
func NewCreateURL() domain.CreateURL {
	return domain.CreateURL{
		ID:             StringPointer("test123"),
		Link:           "http://www.example.org",
		ExpirationDate: DatePointer(time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC()),
		UserID:         "507f191e810c19729de860ea",
	}
}

// NewUpdateURL creates instance of UpdateURL model
func NewUpdateURL() domain.UpdateURL {
	return domain.UpdateURL{
		ID:             "test123",
		ExpirationDate: time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC(),
	}
}

// NewURLBsonD creates instance of User model in bson format
func NewURLBsonD() bson.D {
	return bson.D{
		{Key: "_id", Value: "test123"},
		{Key: "link", Value: "http://www.example.org"},
		{Key: "expiration_date", Value: time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC()},
		{Key: "user_id", Value: "507f191e810c19729de860ea"},
		{Key: "created_at", Value: time.Now().Truncate(time.Millisecond).UTC()},
		{Key: "updated_at", Value: time.Now().Truncate(time.Millisecond).UTC()},
	}
}
