package store

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/semka95/shortener/backend/domain"
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

	if cfg.User == "" || cfg.Password == "" {
		uri.User = nil
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

// StatusHandler represent the http handler for status check
type StatusHandler struct {
	DB *mongo.Database
}

// NewStatusHandler will initialize the /status endpoint
func NewStatusHandler(e *echo.Echo, db *mongo.Database) {
	handler := &StatusHandler{
		DB: db,
	}

	e.GET("/v1/status", handler.StatusCheckHandler)
}

// StatusCheckHandler will get status of the database
func (h *StatusHandler) StatusCheckHandler(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := StatusCheck(ctx, h.DB)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, domain.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

// StatusCheck gets database status and metrics
func StatusCheck(ctx context.Context, db *mongo.Database) (*bson.M, error) {
	statCmd := bson.D{
		primitive.E{Key: "serverStatus", Value: 1},
		primitive.E{Key: "metrics", Value: 1},
	}

	result := new(bson.M)
	if err := db.RunCommand(ctx, statCmd).Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}

// StructToDoc transforms any struct to bson.D document
func StructToDoc(v interface{}) (doc *bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		return doc, err
	}

	err = bson.Unmarshal(data, &doc)
	return doc, err
}
