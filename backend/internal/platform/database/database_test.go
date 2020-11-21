package database_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStatusCheck(t *testing.T) {
	cfg := database.MongoConfig{
		Name:     "test",
		User:     "",
		Password: "",
		HostPort: "localhost:27017",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	logger := zap.NewNop()

	client, err := database.Open(ctx, cfg, logger)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/v1/status", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/v1/status")

	handler := database.StatusHandler{
		DB: client.Database(cfg.Name),
	}

	err = handler.StatusCheckHandler(c)
	assert.NoError(t, err)

	body := make(map[string]interface{})
	err = json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
}

func TestDatabase__ConnectionError(t *testing.T) {
	cfg := database.MongoConfig{
		Name:     "test",
		User:     "",
		Password: "",
		HostPort: "localhost:1234",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	_, err = database.Open(ctx, cfg, logger)
	assert.EqualError(t, err, "ping error: context deadline exceeded")
}
