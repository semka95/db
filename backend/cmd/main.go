package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	_URLHttpDelivery "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	_URLRepo "bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	_URLUcase "bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
)

func init() {
	viper.SetConfigFile(`config.json`)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	timeoutContext := time.Duration(viper.GetInt("context.timeout")) * time.Second
	dbPort := viper.GetString(`mongo.port`)
	dbUser := viper.GetString(`mongo.user`)
	dbPass := viper.GetString(`mongo.pwd`)
	dbName := viper.GetString(`mongo.name`)
	uri := fmt.Sprintf("mongodb://%s:%s@mongodb:%s", dbUser, dbPass, dbPort)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Can't create logger: ", err)
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Println("Can't close logger: ", err)
		}
	}()

	e := echo.New()
	middL := middleware.InitMiddleware(logger)
	e.Use(middL.CORS)
	e.Use(middL.Logger)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		logger.Fatal("MongoDB connection problem: ", zap.Error(err))
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal("Program exit: ", err)
		}
		cancel()
	}()

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		logger.Fatal("Ping error: ", zap.Error(err))
	}
	logger.Info("MongoDB ping: ok")

	ur := _URLRepo.NewMongoURLRepository(client, dbName)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	err = _URLHttpDelivery.NewURLHandler(e, uu)
	if err != nil {
		logger.Fatal("URL handler creation failed: ", zap.Error(err))
	}

	logger.Fatal("Start server error: ", zap.Error(e.Start(viper.GetString("server.address"))))
}
