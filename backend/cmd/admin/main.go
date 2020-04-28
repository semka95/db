package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"bitbucket.org/dbproject_ivt/db/backend/internal/schema"
	"github.com/golang-migrate/migrate/v4"
	dStub "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Config stores app configuration
type Config struct {
	Server struct {
		Address string `yaml:"address"`
		Timeout int    `yaml:"timeout"`
	} `yaml:"server"`
	database.MongoConfig `yaml:"mongo"`
}

func main() {
	// Logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Println("can't create logger: ", err)
		os.Exit(1)
	}
	// defer func() {
	// 	err := logger.Sync()
	// 	if err != nil {
	// 		log.Println("can't close logger: ", err)
	// 		os.Exit(1)
	// 	}
	// }()

	if err := run(logger); err != nil {
		logger.Error("shutting down, error: ", zap.Error(err))
		os.Exit(1)
	}
}

func run(logger *zap.Logger) error {
	f, err := os.Open("../../config.yaml")
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

	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	client, err := database.Open(ctx, cfg.MongoConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logger.Error("mongodb client disconnect error: ", zap.Error(err))
		}
	}()

	switch os.Args[1] {
	case "migrate_mongo":
		err = migrateMongo(client, cfg.MongoConfig.Name)
	case "seed":
		err = schema.Seed(ctx, client.Database(cfg.MongoConfig.Name))
	default:
		err = errors.New("Must specify a command")
	}

	if err != nil {
		return err
	}

	return nil
}

func migrateMongo(db *mongo.Client, dbName string) error {
	instance, err := dStub.WithInstance(db, &dStub.Config{DatabaseName: dbName})
	if err != nil {
		return err
	}

	// change this to something else
	m, err := migrate.NewWithDatabaseInstance("file:///home/semyonz/Документы/db/backend/internal/schema/migrations", dbName, instance)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
