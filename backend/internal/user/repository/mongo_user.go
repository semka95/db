package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
)

type mongoUserRepository struct {
	Conn   *mongo.Database
	logger *zap.Logger
}

// NewMongoUserRepository will create an object that represent the user.Repository interface
func NewMongoUserRepository(c *mongo.Client, db string, logger *zap.Logger) user.Repository {
	return &mongoUserRepository{
		Conn:   c.Database(db),
		logger: logger,
	}
}

func (m *mongoUserRepository) fetch(ctx context.Context, command interface{}) ([]*models.User, error) {
	cur, err := m.Conn.RunCommandCursor(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("Can't execute command: %w", err)
	}

	defer func(ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			m.logger.Error("Can't close cursor: ", zap.Error(err))
		}
	}(ctx)

	result := make([]*models.User, 0)

	for cur.Next(ctx) {
		elem := new(models.User)
		if err := cur.Decode(elem); err != nil {
			return nil, fmt.Errorf("Can't unmarshal document into User: %w", err)
		}

		result = append(result, elem)
	}

	if err = cur.Err(); err != nil {
		return nil, fmt.Errorf("User cursor error: %w", err)
	}

	return result, nil
}

func (m *mongoUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("User ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
	}

	command := bson.D{
		primitive.E{Key: "find", Value: "user"},
		primitive.E{Key: "limit", Value: 1},
		primitive.E{Key: "filter", Value: bson.D{primitive.E{Key: "_id", Value: objID}}},
	}

	list, err := m.fetch(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("User get error: %w: %s", models.ErrInternalServerError, err.Error())
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("User was not found: %w", models.ErrNotFound)
	}

	return list[0], nil
}

func (m *mongoUserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now().Truncate(time.Millisecond).UTC()
	_, err := m.Conn.Collection("user").InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("User store error: %w: %s", models.ErrInternalServerError, err.Error())
	}

	return nil
}

func (m *mongoUserRepository) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("User ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
	}
	filter := bson.D{
		primitive.E{Key: "_id", Value: objID},
	}

	delRes, err := m.Conn.Collection("user").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("User delete error: %w: %s", models.ErrInternalServerError, err.Error())
	}

	if delRes.DeletedCount == 0 {
		return fmt.Errorf("User was not deleted: %w", models.ErrNoAffected)
	}

	return nil
}
func (m *mongoUserRepository) Update(ctx context.Context, user *models.User) error {
	filter := bson.D{
		primitive.E{Key: "_id", Value: user.ID},
	}

	doc, err := toDoc(&user)
	if err != nil {
		return fmt.Errorf("Can't convert User to bson.D: %w, %s", models.ErrInternalServerError, err.Error())
	}
	update := bson.D{primitive.E{Key: "$set", Value: doc}}

	updRes, err := m.Conn.Collection("user").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("User update error: %w: %s", models.ErrInternalServerError, err.Error())
	}

	if updRes.ModifiedCount == 0 {
		return fmt.Errorf("User was not updated: %w", models.ErrNoAffected)
	}

	return nil
}

func toDoc(v interface{}) (doc *bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		return
	}

	err = bson.Unmarshal(data, &doc)
	return
}
