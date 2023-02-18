package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/tests"
	"github.com/semka95/shortener/backend/user/repository"
)

var tracer = sdktrace.NewTracerProvider().Tracer("")
var noopCtx = context.Background()

const tableName = "shortener.user"

//nolint:dupl // test getbyid and getbyemail separately
func TestMongoUserRepository_GetByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tUser := tests.NewUser()
	tUserBsonD := tests.NewUserBsonD()

	mt.Run("not exists", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, tUser.ID)

		assert.Nil(mt, result)
		assert.ErrorIs(mt, err, domain.ErrNotFound)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch, tUserBsonD),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, tUser.ID)

		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, tUser.ID)

		assert.Nil(mt, result)
		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoUserRepository_Create(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tUser := tests.NewUser()

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Create(noopCtx, tUser)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Create(noopCtx, tUser)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoUserRepository_Delete(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tUser := tests.NewUser()

	mt.Run("not found", func(mt *mtest.T) {
		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "acknowledged", Value: true},
				{Key: "n", Value: 0},
			},
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, tUser.ID)

		assert.ErrorIs(mt, err, domain.ErrNoAffected)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "acknowledged", Value: true},
				{Key: "n", Value: 1},
			},
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, tUser.ID)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, tUser.ID)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoUserRepository_Update(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tUser := tests.NewUser()
	tUserBsonD := tests.NewUserBsonD()

	mt.Run("not exists", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 0},
		})
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tUser)

		assert.Error(mt, err, domain.ErrNoAffected)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "value", Value: tUserBsonD},
			{Key: "nModified", Value: 1},
		})
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tUser)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tUser)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

//nolint:dupl // test getbyid and getbyemail separately
func TestMongoUserRepository_GetByEmail(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tUser := tests.NewUser()
	tUserBsonD := tests.NewUserBsonD()

	mt.Run("not exists", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByEmail(noopCtx, tUser.Email)

		assert.Nil(mt, result)
		assert.ErrorIs(mt, err, domain.ErrNotFound)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch, tUserBsonD),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByEmail(noopCtx, tUser.Email)

		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByEmail(noopCtx, tUser.Email)

		assert.Nil(mt, result)
		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}
