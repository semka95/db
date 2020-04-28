package schema

import (
	"context"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Seed inserts data in database for development purposes
func Seed(ctx context.Context, db *mongo.Database) error {
	collections := make(map[string][]interface{}, 2)
	collections["url"] = []interface{}{
		models.URL{
			ID:             "google",
			Link:           "https://www.google.com",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		models.URL{
			ID:             "youtube",
			Link:           "https://www.youtube.com",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		models.URL{
			ID:             "github",
			Link:           "https://www.github.com",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		models.URL{
			ID:             "telegram",
			Link:           "https://www.telegram.org",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		models.URL{
			ID:             "habr",
			Link:           "https://www.habr.com",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		models.URL{
			ID:             "wiki",
			Link:           "https://www.wikipedia.org",
			ExpirationDate: time.Now().Add(time.Hour),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	collections["user"] = []interface{}{
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 1",
			Email:     "test1@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 2",
			Email:     "test2@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 3",
			Email:     "test3@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 4",
			Email:     "test4@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 5",
			Email:     "test5@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		models.User{
			ID:        primitive.NewObjectID(),
			FullName:  "User 6",
			Email:     "test6@example.org",
			Password:  "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for k, v := range collections {
		res, err := db.Collection(k).InsertMany(ctx, v)
		if err != nil || len(res.InsertedIDs) == 0 {
			return err
		}
	}

	return nil
}
