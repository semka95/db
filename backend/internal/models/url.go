package models

import "time"

// URL represent the URL model
type URL struct {
	ID             string    `json:"_id" bson:"_id" validate:"omitempty,linkid,min=7,max=20"`
	Link           string    `json:"link" bson:"link" validate:"required,url,min=4"`
	ExpirationDate time.Time `json:"expiration_date" bson:"expiration_date"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}

// NewURL creates instance of URL model
func NewURL() *URL {
	return &URL{
		ID:             "test123",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC(),
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}
}
