package repository_test

import (
	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"os"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = sdktrace.NewTracerProvider().Tracer("")

func TestMain(m *testing.M) {
	if err := mtest.Setup(); err != nil {
		log.Fatal(err)
	}
	defer os.Exit(m.Run())
	if err := mtest.Teardown(); err != nil {
		log.Fatal(err)
	}
}

func TestMongoURLRepository_GetByID(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.Run("url not exists", func(mt *mtest.T) {
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)
		result, err := r.GetByID(mtest.Background, "none")
		assert.Nil(mt, result)
		require.Error(mt, err, web.ErrNotFound)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)

		result, err := r.GetByID(mtest.Background, tURL.ID)
		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})
}

func TestMongoURLRepository_Store(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.RunOpts("success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Store(mtest.Background, tURL)
		require.NoError(mt, err)

		result := &models.URL{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tURL.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})
}

func TestMongoURLRepository_Delete(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.RunOpts("url not found", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Delete(mtest.Background, "none")
		require.Error(mt, err, web.ErrNoAffected)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err = r.Delete(mtest.Background, tURL.ID)
		require.NoError(mt, err)
	})
}

func TestMongoURLRepository_Update(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.RunOpts("url not exists", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Update(mtest.Background, tURL)
		require.Error(mt, err, web.ErrNoAffected)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)
		r := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil, tracer)

		tURL.Link = "https://www.google.com"
		err = r.Update(mtest.Background, tURL)
		require.NoError(mt, err)

		result := &models.URL{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tURL.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})
}
