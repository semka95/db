package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/config"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"bitbucket.org/dbproject_ivt/db/backend/internal/schema"
	"github.com/golang-migrate/migrate/v4"
	dStub "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

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
	configPath, ok := os.LookupEnv("CONFIG")
	if !ok {
		return fmt.Errorf("CONFIG environment variable is not specified")
	}
	cfg, err := config.AppConfig(configPath, logger)
	if err != nil {
		return err
	}

	timeoutContext := time.Duration(cfg.Server.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutContext)
	defer cancel()

	// Start database
	cfg.MongoConfig.HostPort = "localhost:27017"
	client, err := database.Open(ctx, cfg.MongoConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logger.Error("mongodb client disconnect error: ", zap.Error(err))
		}
	}()

	if len(os.Args) < 2 {
		return errors.New("must specify a command")
	}

	switch os.Args[1] {
	case "migrate_mongo":
		err = migrateMongo(client, cfg.MongoConfig.Name)
	case "seed":
		err = schema.Seed(ctx, client.Database(cfg.MongoConfig.Name))
	case "keygen":
		err = keygen(os.Args[2], logger)
	default:
		err = errors.New("must specify a command")
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
	m, err := migrate.NewWithDatabaseInstance("file://./internal/schema/migrations", dbName, instance)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// keygen creates an x509 private key for signing auth tokens.
func keygen(path string, logger *zap.Logger) error {
	if path == "" {
		return errors.New("keygen missing argument for key path")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating private file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			logger.Error("can't close .pem file: ", zap.Error(err))
		}
	}()

	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	if err := pem.Encode(file, &block); err != nil {
		return fmt.Errorf("encoding to private file: %w", err)
	}

	return nil
}
