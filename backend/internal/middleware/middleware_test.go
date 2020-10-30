package middleware_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_mid "bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
)

func TestCORS(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)
	m := _mid.InitMiddleware(nil)

	h := m.CORS(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

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

	m := _mid.InitMiddleware(logger)

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
				return errors.New("test error")
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
	require.NoError(t, err)

	expClaims := auth.NewClaims("test user", []string{auth.RoleUser}, time.Now().Add(-time.Hour), time.Minute)
	expToken, err := authenticator.GenerateToken(expClaims)
	require.NoError(t, err)

	cases := []struct {
		Description string
		Token       string
		Code        int
		Message     string
		Success     bool
	}{
		{
			"auth success",
			token,
			0,
			"",
			true,
		},
		{
			"auth token expired",
			expToken,
			http.StatusUnauthorized,
			"invalid or expired jwt",
			false,
		},
		{
			"auth token not provided",
			"",
			http.StatusBadRequest,
			"missing or malformed jwt",
			false,
		},
	}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(echo.GET, "/", nil)
			req.Header.Set("Authorization", "Bearer "+test.Token)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			m := middleware.JWTWithConfig(authenticator.JWTConfig)

			h := m(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			err = h(c)
			if test.Success {
				require.NoError(t, err)
				return
			}

			var he *echo.HTTPError
			if !errors.As(err, &he) {
				t.Error("error is not type of echo.HTTPError")
			}
			assert.Equal(t, test.Code, he.Code)
			assert.Equal(t, test.Message, he.Message)
		})
	}
}

func TestHasRole(t *testing.T) {
	claims := auth.NewClaims("test user", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t.Run("role check success", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		c.Set("user", token)

		m := _mid.InitMiddleware(nil).HasRole(auth.RoleUser)
		h := m(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := h(c)
		require.NoError(t, err)
	})

	t.Run("role check fail", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		c.Set("user", token)

		m := _mid.InitMiddleware(nil).HasRole(auth.RoleAdmin)
		h := m(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := h(c)
		var herr *echo.HTTPError
		if !errors.As(err, &herr) {
			t.Error("error is not type of echo.HTTPError")
		}
		assert.EqualValues(t, herr.Message, "you are not authorized for that action")
	})
}
