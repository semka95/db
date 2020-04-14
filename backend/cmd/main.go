package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	_URLHttpDelivery "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	_URLRepo "bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	_URLUcase "bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
)

// Config stores app configuration
type Config struct {
	Server struct {
		Address string `yaml:"address,required"`
		Timeout int    `yaml:"timeout,required"`
	} `yaml:"server,required"`
	Mongo struct {
		Name     string `yaml:"name,required"`
		User     string `yaml:"user,required"`
		Password string `yaml:"pwd,required"`
		Port     string `yaml:"port,required"`
	} `yaml:"mongo,required"`
}

func main() {
	f, err := os.Open("../config.yaml")
	if err != nil {
		log.Fatal("Can't open config file: ", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println("Can't close config file: ", err)
		}
	}()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Can't decode config file: ", err)
	}

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

	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	uri := fmt.Sprintf("mongodb://%s:%s@mongodb:%s", cfg.Mongo.User, cfg.Mongo.Password, cfg.Mongo.Port)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)

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

	ur := _URLRepo.NewMongoURLRepository(client, cfg.Mongo.Name, logger)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	err = _URLHttpDelivery.NewURLHandler(e, uu, logger)
	if err != nil {
		logger.Fatal("URL handler creation failed: ", zap.Error(err))
	}

	logger.Fatal("Start server error: ", zap.Error(e.Start(cfg.Server.Address)))
}
