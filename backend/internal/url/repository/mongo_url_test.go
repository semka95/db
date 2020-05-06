package repository_test

import (
	"log"
	"os"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

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

	mt.Run("get url not exist", func(mt *mtest.T) {
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, "none")
		assert.Nil(mt, result)
		require.Error(mt, err, web.ErrNotFound)
	})

	mt.RunOpts("get url success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)

		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, tURL.ID)
		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})
}

func TestMongoURLRepository_Store(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.RunOpts("store url success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Store(mtest.Background, tURL)
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

	mt.RunOpts("delete not existing url", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Delete(mtest.Background, "none")
		require.Error(mt, err, web.ErrNoAffected)
	})

	mt.RunOpts("delete url success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)

		err = repository.Delete(mtest.Background, tURL.ID)
		require.NoError(mt, err)
	})
}

func TestMongoURLRepository_Update(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tURL := tests.NewURL()

	mt.RunOpts("update not existing url", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Update(mtest.Background, tURL)
		require.Error(mt, err, web.ErrNoAffected)
	})

	mt.RunOpts("update url success", mtest.NewOptions().CollectionName("url"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tURL)
		require.NoError(mt, err)
		repository := repository.NewMongoURLRepository(mt.Client, mt.DB.Name(), nil)

		tURL.Link = "https://www.google.com"
		err = repository.Update(mtest.Background, tURL)
		require.NoError(mt, err)

		result := &models.URL{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tURL.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tURL, result)
	})
}
