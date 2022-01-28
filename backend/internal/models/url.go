package models

import "time"

// URL represents the URL model
type URL struct {
	ID             string    `json:"id" bson:"_id"`
	Link           string    `json:"link" bson:"link"`
	ExpirationDate time.Time `json:"expiration_date" bson:"expiration_date"`
	UserID         string    `json:"user_id" bson:"user_id"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

// CreateURL represents data to create new URL
type CreateURL struct {
	ID             *string    `json:"id" validate:"omitempty,linkid,min=7,max=20"`
	Link           string     `json:"link" validate:"required,url"`
	ExpirationDate *time.Time `json:"expiration_date" validate:"omitempty,gt"`
	UserID         string     `json:"-"`
}

// UpdateURL represents data to update URL
type UpdateURL struct {
	ID             string    `json:"id" validate:"required,linkid,max=20"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required,gt"`
}
