package domain

import (
	"context"
	"time"

	"github.com/semka95/shortener/backend/web/auth"
)

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

// URLUsecase represents the URL's usecases
type URLUsecase interface {
	GetByID(ctx context.Context, id string) (*URL, error)
	Update(ctx context.Context, updateURL UpdateURL, user auth.Claims) error
	Store(ctx context.Context, createURL CreateURL) (*URL, error)
	Delete(ctx context.Context, id string, user auth.Claims) error
}

// URLRepository represents the URL's repository contract
type URLRepository interface {
	GetByID(ctx context.Context, id string) (*URL, error)
	Update(ctx context.Context, url *URL) error
	Store(ctx context.Context, u *URL) error
	Delete(ctx context.Context, id string) error
}
