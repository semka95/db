package repository_test

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/tests"
	"github.com/semka95/shortener/backend/user/repository"
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

func TestMongoUserRepository_GetByID(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("user not exist", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		result, err := r.GetByID(mtest.Background, tUser.ID)
		assert.Nil(mt, result)
		assert.Error(mt, err, domain.ErrNotFound)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		assert.NoError(mt, err)

		result, err := r.GetByID(mtest.Background, tUser.ID)
		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_Create(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Create(mtest.Background, tUser)
		require.NoError(mt, err)

		result := &domain.User{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tUser.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_Delete(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("user not found", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Delete(mtest.Background, tUser.ID)
		assert.Error(mt, err, domain.ErrNoAffected)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		require.NoError(mt, err)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		err = r.Delete(mtest.Background, tUser.ID)
		require.NoError(mt, err)
	})
}

func TestMongoUserRepository_Update(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("user not exists", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		err := r.Update(mtest.Background, tUser)
		assert.Error(mt, err, domain.ErrNoAffected)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		require.NoError(mt, err)
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)

		tUser.FullName = "Test User"
		tUser.Email = "123@test.org"
		err = r.Update(mtest.Background, tUser)
		require.NoError(mt, err)

		result := &domain.User{}
		err = mt.Coll.FindOne(mtest.Background, bson.D{primitive.E{Key: "_id", Value: tUser.ID}}).Decode(result)
		require.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}

func TestMongoUserRepository_GetByEmail(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()
	tUser := tests.NewUser()

	mt.RunOpts("user not exists", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		result, err := r.GetByEmail(mtest.Background, tUser.Email)
		assert.Nil(mt, result)
		assert.Error(mt, err, domain.ErrNotFound)
	})

	mt.RunOpts("success", mtest.NewOptions().CollectionName("user"), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(mtest.Background, tUser)
		assert.NoError(mt, err)

		r := repository.NewMongoUserRepository(mt.Client, mt.DB.Name(), nil, tracer)
		result, err := r.GetByEmail(mtest.Background, tUser.Email)
		assert.NoError(mt, err)
		assert.EqualValues(t, tUser, result)
	})
}
