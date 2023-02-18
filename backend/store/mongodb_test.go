package store_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"github.com/semka95/shortener/backend/store"
)

func TestStatusCheck(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/v1/status", nil)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/v1/status")

		handler := store.StatusHandler{
			DB: mt.Client.Database("shortener"),
		}

		err := handler.StatusCheckHandler(c)
		assert.NoError(t, err)

		body := make(map[string]interface{})
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)

	})
	mt.Run("error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    1,
			Message: "test",
			Name:    "123",
			Labels:  []string{},
		}))
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/v1/status", nil)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/v1/status")

		handler := store.StatusHandler{
			DB: mt.Client.Database("shortener"),
		}

		err := handler.StatusCheckHandler(c)
		assert.NoError(t, err)

		body := make(map[string]interface{})
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(mt, "(123) test", body["error"])
	})
}

// func TestDatabase__ConnectionError(t *testing.T) {
// 	cfg := store.MongoConfig{
// 		Name:     "test",
// 		User:     "",
// 		Password: "",
// 		HostPort: "localhost:1234",
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	logger, err := zap.NewDevelopment()
// 	require.NoError(t, err)

// 	_, err = store.Open(ctx, cfg, logger)
// 	assert.Contains(t, err.Error(), "ping error")
// }
