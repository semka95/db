package database

import (
	"context"
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// MongoConfig stores MongoDB configuration
type MongoConfig struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"pwd"`
	HostPort string `yaml:"host_port"`
}

// Open creates MongoDB client
func Open(ctx context.Context, cfg MongoConfig, logger *zap.Logger) (*mongo.Client, error) {
	uri := url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   cfg.HostPort,
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri.String()))
	if err != nil {
		return nil, fmt.Errorf("mongodb connection problem: %w", err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("ping error: %w", err)
	}
	logger.Info("mongodb ping: ok")

	return client, nil
}
