package models

import "time"

// URL represent the URL model
type URL struct {
	ID             string    `json:"_id" bson:"_id" validate:"omitempty,linkid,min=7,max=20"`
	Link           string    `json:"link" bson:"link" validate:"required,url,min=4"`
	ExpirationDate time.Time `json:"expiration_date" bson:"expiration_date"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}
