package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
)

type mongoUserRepository struct {
	Conn   *mongo.Database
	logger *zap.Logger
	tracer trace.Tracer
}

// NewMongoUserRepository will create an object that represent the user.Repository interface
func NewMongoUserRepository(c *mongo.Client, db string, logger *zap.Logger, tracer trace.Tracer) user.Repository {
	return &mongoUserRepository{
		Conn:   c.Database(db),
		logger: logger,
		tracer: tracer,
	}
}

func (m *mongoUserRepository) fetch(ctx context.Context, command interface{}) ([]*models.User, error) {
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
			m.logger.Error("Can't close cursor: ", zap.Error(err))
		}
	}(ctx)

	result := make([]*models.User, 0)

	for cur.Next(ctx) {
		elem := new(models.User)
		if err := cur.Decode(elem); err != nil {
			span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
			return nil, fmt.Errorf("can't unmarshal document into User: %w", err)
		}

		result = append(result, elem)
	}

	if err = cur.Err(); err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("user cursor error: %w", err)
	}

	return result, nil
}

func (m *mongoUserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	ctx, span := m.tracer.Start(
		ctx,
		"repository GetByID",
		trace.WithAttributes(
			label.String("userid", id.Hex())),
	)
	defer span.End()

	command := bson.D{
		primitive.E{Key: "find", Value: "user"},
		primitive.E{Key: "limit", Value: 1},
		primitive.E{Key: "filter", Value: bson.D{primitive.E{Key: "_id", Value: id}}},
	}

	list, err := m.fetch(ctx, command)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("user get error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if len(list) == 0 {
		span.RecordError(ctx, web.ErrNotFound, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("user was not found: %w", web.ErrNotFound)
	}

	return list[0], nil
}

func (m *mongoUserRepository) Create(ctx context.Context, user *models.User) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Create",
		trace.WithAttributes(
			label.String("userid", user.ID.Hex())),
	)
	defer span.End()

	_, err := m.Conn.Collection("user").InsertOne(ctx, user)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("user store error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	return nil
}

func (m *mongoUserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Delete",
		trace.WithAttributes(
			label.String("userid", id.Hex())),
	)
	defer span.End()

	filter := bson.D{
		primitive.E{Key: "_id", Value: id},
	}

	delRes, err := m.Conn.Collection("user").DeleteOne(ctx, filter)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("user delete error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if delRes.DeletedCount == 0 {
		err = fmt.Errorf("user was not deleted: %w", web.ErrNoAffected)
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return err
	}

	return nil
}
func (m *mongoUserRepository) Update(ctx context.Context, user *models.User) error {
	ctx, span := m.tracer.Start(
		ctx,
		"repository Update",
		trace.WithAttributes(
			label.String("userid", user.ID.Hex())),
	)
	defer span.End()

	filter := bson.D{
		primitive.E{Key: "_id", Value: user.ID},
	}

	doc, err := database.StructToDoc(&user)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("can't convert User to bson.D: %w, %s", web.ErrInternalServerError, err.Error())
	}
	update := bson.D{primitive.E{Key: "$set", Value: doc}}

	updRes, err := m.Conn.Collection("user").UpdateOne(ctx, filter, update)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return fmt.Errorf("user update error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if updRes.ModifiedCount == 0 {
		err = fmt.Errorf("user was not updated: %w", web.ErrNoAffected)
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return err
	}

	return nil
}

func (m *mongoUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, span := m.tracer.Start(
		ctx,
		"repository GetByEmail",
	)
	defer span.End()

	command := bson.D{
		primitive.E{Key: "find", Value: "user"},
		primitive.E{Key: "limit", Value: 1},
		primitive.E{Key: "filter", Value: bson.D{primitive.E{Key: "email", Value: email}}},
	}

	list, err := m.fetch(ctx, command)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("user get error: %w: %s", web.ErrInternalServerError, err.Error())
	}

	if len(list) == 0 {
		span.RecordError(ctx, web.ErrNotFound, trace.WithErrorStatus(codes.Error))
		return nil, fmt.Errorf("user with email %s was not found: %w", email, web.ErrNotFound)
	}

	span.SetAttributes(label.String("userid", list[0].ID.Hex()))

	return list[0], nil
}
