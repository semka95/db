package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/web/auth"
)

// Seed inserts data in database for development purposes
func Seed(ctx context.Context, db *mongo.Database) error {
	collections := make(map[string][]interface{}, 2)
	timeNow := time.Now().Truncate(time.Millisecond).UTC()
	expTime := time.Now().Add(time.Hour).Truncate(time.Millisecond).UTC()
	roles := []string{auth.RoleUser}

	collections["url"] = []interface{}{
		domain.URL{
			ID:             "google",
			Link:           "https://www.google.com",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.URL{
			ID:             "youtube",
			Link:           "https://www.youtube.com",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.URL{
			ID:             "github",
			Link:           "https://www.github.com",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.URL{
			ID:             "telegram",
			Link:           "https://www.telegram.org",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.URL{
			ID:             "habr",
			Link:           "https://www.habr.com",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.URL{
			ID:             "wiki",
			Link:           "https://www.wikipedia.org",
			ExpirationDate: expTime,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
	}

	collections["user"] = []interface{}{
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 1",
			Email:          "test1@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 2",
			Email:          "test2@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 3",
			Email:          "test3@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 4",
			Email:          "test4@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 5",
			Email:          "test5@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
		},
		domain.User{
			ID:             primitive.NewObjectID(),
			FullName:       "User 6",
			Email:          "test6@example.org",
			HashedPassword: "$2a$10$2iPnt444yuUBu8tSCm0iXOaGO2YYyTLVzGKr9LudAj7s.9m9iv7PS",
			Roles:          roles,
			CreatedAt:      timeNow,
			UpdatedAt:      timeNow,
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
