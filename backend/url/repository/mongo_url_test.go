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
	"github.com/semka95/shortener/backend/url/repository"
)

var tracer = sdktrace.NewTracerProvider().Tracer("")
var noopCtx = context.Background()

const tableName = "shortener.url"

func TestMongoURLRepository_GetByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tURL := tests.NewURL()
	tURLBsonD := tests.NewURLBsonD()

	mt.Run("not exists", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, "none")

		assert.Nil(mt, result)
		require.Error(mt, err, domain.ErrNotFound)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, tableName, mtest.FirstBatch, tURLBsonD),
			mtest.CreateCursorResponse(0, tableName, mtest.NextBatch),
		)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, tURL.ID)

		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		result, err := r.GetByID(noopCtx, tURL.ID)

		assert.Nil(mt, result)
		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoURLRepository_Store(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tURL := tests.NewURL()

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Store(noopCtx, tURL)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Store(noopCtx, tURL)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoURLRepository_Delete(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tURL := tests.NewURL()

	mt.Run("not found", func(mt *mtest.T) {
		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "acknowledged", Value: true},
				{Key: "n", Value: 0},
			},
		)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, "none")

		require.Error(mt, err, domain.ErrNoAffected)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "acknowledged", Value: true},
				{Key: "n", Value: 1},
			},
		)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, tURL.ID)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Delete(noopCtx, tURL.ID)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}

func TestMongoURLRepository_Update(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	tURL := tests.NewURL()
	tURLBsonD := tests.NewURLBsonD()

	mt.Run("not exists", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "nModified", Value: 0},
		})
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tURL)

		require.Error(mt, err, domain.ErrNoAffected)
	})

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "value", Value: tURLBsonD},
			{Key: "nModified", Value: 1},
		})
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tURL)

		require.NoError(mt, err)
	})

	mt.Run("server error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    123,
			Message: "server error",
		}))
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err := r.Update(noopCtx, tURL)

		assert.ErrorIs(mt, err, domain.ErrInternalServerError)
	})
}
