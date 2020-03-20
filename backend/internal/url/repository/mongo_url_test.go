package repository_test

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	// mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	// defer mt.Close()
	// mt.Run("test", func(mt *mtest.T) {
	// 	err := mt.DB.Client().Connect(mtest.Background)
	// 	if err != nil {
	// 		mt.T.Error("lol")
	// 	}
	// })
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("3.6").Topologies(mtest.ReplicaSet, mtest.Sharded).CreateClient(false))
	defer mt.Close()

	mt.Run("operation time nil", func(mt *mtest.T) {
		// when a ClientSession is first created, the operation time is nil
		log.Println("IM HERE!")
		sess, err := mt.Client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		defer sess.EndSession(mtest.Background)
		assert.Nil(mt, sess.OperationTime(), "expected nil operation time, got %v", sess.OperationTime())
	})
}
