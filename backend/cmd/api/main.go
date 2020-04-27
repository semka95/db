package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
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
		HostPort string `yaml:"host_port,required"`
	} `yaml:"mongo,required"`
}

func main() {
	// Logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Println("can't create logger: ", err)
		os.Exit(1)
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Println("can't close logger: ", err)
			os.Exit(1)
		}
	}()

	if err := run(logger); err != nil {
		logger.Error("shutting down, error: ", zap.Error(err))
		os.Exit(1)
	}
}

func run(logger *zap.Logger) error {
	// Configuration
	f, err := os.Open("../config.yaml")
	if err != nil {
		return fmt.Errorf("can't open config file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Error("can't close config file: %w", zap.Error(err))
		}
	}()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return fmt.Errorf("can't decode config file: %w", err)
	}

	// MongoDB configure
	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	uri := url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(cfg.Mongo.User, cfg.Mongo.Password),
		Host:   cfg.Mongo.HostPort,
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	// Echo configure
	e := echo.New()
	middL := middleware.InitMiddleware(logger)
	e.Use(middL.CORS)
	e.Use(middL.Logger)

	// Start database
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri.String()))
	if err != nil {
		return fmt.Errorf("mongodb connection problem: %w", err)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logger.Error("mongodb client disconnect error: ", zap.Error(err))
		}
	}()

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping error: %w", err)
	}
	logger.Info("mongodb ping: ok")

	// Start service
	ur := _URLRepo.NewMongoURLRepository(client, cfg.Mongo.Name, logger)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	err = _URLHttpDelivery.NewURLHandler(e, uu, logger)
	if err != nil {
		return fmt.Errorf("url handler creation failed: %w", err)
	}

	return e.Start(cfg.Server.Address)
}
