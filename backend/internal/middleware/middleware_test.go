package middleware_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	middl "bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
)

func TestCORS(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)
	m := middl.InitMiddleware(nil)

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

	m := middl.InitMiddleware(logger)

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

func TestJWT(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	require.NoError(t, err)

	kid := "4754d86b-7a6d-4df5-9c65-224741361492"
	kf := auth.NewSimpleKeyLookupFunc(kid, key.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(key, kid, "RS256", kf)
	require.NoError(t, err)

	claims := auth.NewClaims("test user", []string{auth.RoleUser}, time.Now(), time.Minute)
	token, err := authenticator.GenerateToken(claims)
	fmt.Println(token)
	require.NoError(t, err)

	t.Run("auth success", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		m := middleware.JWTWithConfig(authenticator.JWTConfig)

		h := m(echo.HandlerFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		}))

		err = h(c)
		require.NoError(t, err)
	})

	t.Run("auth token expired", func(t *testing.T) {
		expClaims := auth.NewClaims("test user", []string{auth.RoleUser}, time.Now().Add(-time.Hour), time.Minute)
		expToken, err := authenticator.GenerateToken(expClaims)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		req.Header.Set("Authorization", "Bearer "+expToken)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		m := middleware.JWTWithConfig(authenticator.JWTConfig)

		h := m(echo.HandlerFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		}))

		err = h(c)
		assert.Contains(t, err.Error(), "token is expired")
	})

	t.Run("auth token not provided", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		m := middleware.JWTWithConfig(authenticator.JWTConfig)

		h := m(echo.HandlerFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		}))

		err = h(c)
		var herr *echo.HTTPError
		if errors.As(err, &herr) {
			fmt.Println(herr.Message)
		}
		assert.EqualValues(t, herr.Message, "missing or malformed jwt")
	})

	t.Run("auth wrong signature", func(t *testing.T) {
		fakeToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjQ3NTRkODZiLTdhNmQtNGRmNS05YzY1LTIyNDc0MTM2MTQ5MiIsInR5cCI6IkpXVCJ9.eyJSb2xlcyI6WyJVU0VSIl0sImV4cCI6MTU5MTMwMTc2NywiaWF0IjoxNTkxMzAxNzA3LCJzdWIiOiJ0ZXN0IHVzZXIifQ.eyfake"
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		req.Header.Set("Authorization", "Bearer "+fakeToken)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		m := middleware.JWTWithConfig(authenticator.JWTConfig)

		h := m(echo.HandlerFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		}))

		err = h(c)
		assert.Contains(t, err.Error(), "verification error")
	})
}
