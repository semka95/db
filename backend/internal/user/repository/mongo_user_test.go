package repository_test

import (
	"log"
	"os"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
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
	tUser := models.NewUser()

	mt.RunOpts("get user not valid id", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, "not valid")
		assert.Nil(mt, result)
		assert.Error(mt, err, models.ErrBadParamInput)
	})

	mt.RunOpts("get user not exist", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, "507f191e810c19729de860ea")
		assert.Nil(mt, result)
		assert.Error(mt, err, models.ErrNotFound)
	})

	mt.RunOpts("get user success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		assert.NoError(mt, err)

		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		result, err := repository.GetByID(mtest.Background, tUser.ID.Hex())
		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_Create(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := models.NewUser()

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
	tUser := models.NewUser()

	mt.RunOpts("delete user not valid id", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Delete(mtest.Background, "not valid")
		assert.Error(mt, err, models.ErrBadParamInput)
	})

	mt.RunOpts("delete not existing user", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)
		err := repository.Delete(mtest.Background, "507f191e810c19729de860ea")
		assert.Error(mt, err, models.ErrNoAffected)
	})

	mt.RunOpts("delete success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		require.NoError(mt, err)
		repository := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil)

		err = repository.Delete(mtest.Background, tUser.ID.Hex())
		require.NoError(mt, err)
	})
}

func TestMongoUserRepository_Update(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := models.NewUser()

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
