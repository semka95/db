package models

import "time"

// URL represent the URL model
type URL struct {
	ID             string    `json:"_id" bson:"_id"`
	Link           string    `json:"link" bson:"link"`
	ExpirationDate time.Time `json:"expiration_date" bson:"expiration_date"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}
