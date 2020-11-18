package repository

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/codes"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

type mongoURLRepository struct {
	Conn   *mongo.Database
	logger *zap.Logger
	tracer trace.Tracer
}

// NewMongoURLRepository will create an object that represent the url.Repository interface
func NewMongoURLRepository(c *mongo.Client, db string, logger *zap.Logger, tracer trace.Tracer) url.Repository {
	return &mongoURLRepository{
		Conn:   c.Database(db),
		logger: logger,
		tracer: tracer,
	}
}

func (m *mongoURLRepository) fetch(ctx context.Context, command interface{}) ([]*models.URL, error) {
	ctx, span := m.tracer.Start(ctx, "repository fetch")
	defer span.End()

	cur, err := m.Conn.RunCommandCursor(ctx, command)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("can't execute command: %w", err)
	}

	defer func(ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			m.logger.Error("can't close cursor: ", zap.Error(err))
		}
	}(ctx)

	result := make([]*models.URL, 0)

	for cur.Next(ctx) {
		elem := new(models.URL)
		if err := cur.Decode(elem); err != nil {
			span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
			return nil, fmt.Errorf("can't unmarshal document into URL: %w", err)
		}

		result = append(result, elem)
	}

	if err = cur.Err(); err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("URL cursor error: %w", err)
	}

	return result, nil
}

func (m *mongoURLRepository) GetByID(ctx context.Context, id string) (*models.URL, error) {
	ctx, span := m.tracer.Start(
		ctx,
		"repository GetByID",
		trace.WithAttributes(
			label.String("urlid", id)),
	)
	defer span.End()

	command := bson.D{
		primitive.E{Key: "find", Value: "url"},
		primitive.E{Key: "limit", Value: 1},
		primitive.E{Key: "filter", Value: bson.D{primitive.E{Key: "_id", Value: id}}},
	}

	list, err := m.fetch(ctx, command)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("URL get error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if len(list) == 0 {
		span.RecordError(ctx, web.ErrNotFound, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("URL was not found: %w", web.ErrNotFound)
	}

	return list[0], nil
}

func (m *mongoURLRepository) Store(ctx context.Context, url *models.URL) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Store",
		trace.WithAttributes(
			label.String("urlid", url.ID)),
	)
	defer span.End()

	_, err := m.Conn.Collection("url").InsertOne(ctx, url)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("URL store error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	return nil
}

func (m *mongoURLRepository) Delete(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Delete",
		trace.WithAttributes(
			label.String("urlid", id)),
	)
	defer span.End()

	filter := bson.D{
		primitive.E{Key: "_id", Value: id},
	}

	delRes, err := m.Conn.Collection("url").DeleteOne(ctx, filter)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("URL delete error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if delRes.DeletedCount == 0 {
		err = fmt.Errorf("URL was not deleted: %w", web.ErrNoAffected)
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return err
	}

	return nil
}

func (m *mongoURLRepository) Update(ctx context.Context, url *models.URL) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Update",
		trace.WithAttributes(
			label.String("urlid", url.ID)),
	)
	defer span.End()

	filter := bson.D{
		primitive.E{Key: "_id", Value: url.ID},
	}

	doc, err := toDoc(&url)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("can't convert URL to bson.D: %w, %s", web.ErrInternalServerError, err.Error())
	}
	update := bson.D{primitive.E{Key: "$set", Value: doc}}

	updRes, err := m.Conn.Collection("url").UpdateOne(ctx, filter, update)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("URL update error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if updRes.ModifiedCount == 0 {
		err = fmt.Errorf("URL was not updated: %w", web.ErrNoAffected)
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return err
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
