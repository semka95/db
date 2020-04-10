package middleware_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
)

func TestCORS(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)
	m := middleware.InitMiddleware(nil)

	h := m.CORS(echo.HandlerFunc(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}))

	err := h(c)
	require.NoError(t, err)
	assert.Equal(t, "*", res.Header().Get("Access-Control-Allow-Origin"))
}

type loggerJSON struct {
	Level   string `json:"L"`
	Message string `json:"M"`
	Status  int    `json:"status"`
	Method  string `json:"method"`
	URI     string `json:"uri"`
}

func TestLogger(t *testing.T) {
	var b []byte
	l := bytes.NewBuffer(b)
	writerSyncer := zapcore.AddSync(l)
	encoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)
	logger := zap.New(core)
	defer func() {
		err := logger.Sync()
		if err != nil {
			t.Log("Can't close logger")
		}
	}()

	m := middleware.InitMiddleware(logger)

	cases := []struct {
		Description string
		MidFunc     echo.HandlerFunc
		Want        loggerJSON
	}{
		{
			"test success",
			echo.HandlerFunc(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			}),
			loggerJSON{Level: "INFO", Message: "Success", Status: 200, Method: "GET", URI: "/"},
		},
		{
			"test server error",
			echo.HandlerFunc(func(c echo.Context) error {
				return c.NoContent(http.StatusInternalServerError)
			}),
			loggerJSON{Level: "ERROR", Message: "Server error", Status: 500, Method: "GET", URI: "/"},
		},
		{
			"test client error",
			echo.HandlerFunc(func(c echo.Context) error {
				return c.NoContent(http.StatusBadRequest)
			}),
			loggerJSON{Level: "WARN", Message: "Client error", Status: 400, Method: "GET", URI: "/"},
		},
		{
			"test redirection",
			echo.HandlerFunc(func(c echo.Context) error {
				return c.NoContent(http.StatusMovedPermanently)
			}),
			loggerJSON{Level: "INFO", Message: "Redirection", Status: 301, Method: "GET", URI: "/"},
		},
	}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(echo.GET, "/", nil)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)

			h := m.Logger(test.MidFunc)
			err := h(c)
			require.NoError(t, err)

			answer := new(loggerJSON)
			err = json.Unmarshal(l.Bytes(), answer)
			require.NoError(t, err)

			assert.EqualValues(t, test.Want, *answer)

			l.Reset()
		})
	}

	t.Run("test error handler", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		h := m.Logger(echo.HandlerFunc(func(c echo.Context) error {
			return fmt.Errorf("test error")
		}))

		err := h(c)
		require.NoError(t, err)
		got := make(map[string]string)
		err = json.Unmarshal(res.Body.Bytes(), &got)
		require.NoError(t, err)

		assert.Equal(t, got["message"], "Internal Server Error")
	})
}
