package repository

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

type mongoURLRepository struct {
	Conn *mongo.Database
}

// NewMongoURLRepository will create an object that represent the url.Repository interface
func NewMongoURLRepository(c *mongo.Client, db string) url.Repository {
	return &mongoURLRepository{c.Database(db)}
}

func (m *mongoURLRepository) fetch(ctx context.Context, command interface{}) ([]*models.URL, error) {
	cur, err := m.Conn.RunCommandCursor(ctx, command)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	defer func(ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			logrus.Error(err)
		}
	}(ctx)

	result := make([]*models.URL, 0)

	for cur.Next(ctx) {
		elem := new(models.URL)
		if err := cur.Decode(elem); err != nil {
			logrus.Error(err)
			return nil, err
		}

		result = append(result, elem)
	}

	if err = cur.Err(); err != nil {
		logrus.Error(err)
		return nil, err
	}

	return result, nil
}

func (m *mongoURLRepository) GetByID(ctx context.Context, id string) (*models.URL, error) {
	command := bson.D{
		primitive.E{Key: "find", Value: "url"},
		primitive.E{Key: "limit", Value: 1},
		primitive.E{Key: "filter", Value: bson.D{primitive.E{Key: "_id", Value: id}}},
	}

	list, err := m.fetch(ctx, command)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, models.ErrNotFound
	}

	return list[0], nil
}

func (m *mongoURLRepository) Store(ctx context.Context, url *models.URL) error {
	_, err := m.Conn.Collection("url").InsertOne(ctx, url)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoURLRepository) Delete(ctx context.Context, id string) error {
	filter := bson.D{
		primitive.E{Key: "_id", Value: id},
	}

	delRes, err := m.Conn.Collection("url").DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if delRes.DeletedCount == 0 {
		return models.ErrNoAffected
	}

	return nil
}
func (m *mongoURLRepository) Update(ctx context.Context, url *models.URL) error {
	filter := bson.D{
		primitive.E{Key: "_id", Value: url.ID},
	}

	doc, err := toDoc(&url)
	if err != nil {
		return err
	}
	update := bson.D{primitive.E{Key: "$set", Value: doc}}

	updRes, err := m.Conn.Collection("url").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if updRes.ModifiedCount == 0 {
		return models.ErrNoAffected
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
