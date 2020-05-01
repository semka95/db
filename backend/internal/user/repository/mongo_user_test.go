package repository_test

import (
	"log"
	"os"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user/repository"
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

func TestMongoUserRepository_GetByID(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("get user not exist", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, tUser.ID)
		assert.Nil(mt, result)
		assert.Error(mt, err, models.ErrNotFound)
	})

	mt.RunOpts("get user success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		assert.NoError(mt, err)

		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, tUser.ID)
		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_Create(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("create user success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Create(mtest.Background, tUser)
		require.NoError(mt, err)

		result := &models.User{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tUser.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_Delete(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("delete not existing user", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Delete(mtest.Background, tUser.ID)
		assert.Error(mt, err, models.ErrNoAffected)
	})

	mt.RunOpts("delete success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		require.NoError(mt, err)
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)

		err = repository.Delete(mtest.Background, tUser.ID)
		require.NoError(mt, err)
	})
}

func TestMongoUserRepository_Update(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("update not existing user", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Update(mtest.Background, tUser)
		assert.Error(mt, err, models.ErrNoAffected)
	})

	mt.RunOpts("update user success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		require.NoError(mt, err)
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)

		tUser.FullName = "Test User"
		tUser.Email = "123@test.org"
		err = repository.Update(mtest.Background, tUser)
		require.NoError(mt, err)

		result := &models.User{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tUser.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}
