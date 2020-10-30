package middleware

import (
	"net/http"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GoMiddleware represent the data-struct for middleware
type GoMiddleware struct {
	logger *zap.Logger
}

// InitMiddleware initialize the middleware
func InitMiddleware(logger *zap.Logger) *GoMiddleware {
	return &GoMiddleware{
		logger: logger,
	}
}

// CORS will handle the CORS middleware
func (m *GoMiddleware) CORS(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		return next(c)
	}
}

// Logger is a middleware that logs requests
func (m *GoMiddleware) Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		err := next(c)
		if err != nil {
			c.Error(err)
		}

		req := c.Request()
		res := c.Response()

		id := req.Header.Get(echo.HeaderXRequestID)
		if id == "" {
			id = res.Header().Get(echo.HeaderXRequestID)
		}

		fields := []zapcore.Field{
			zap.Int("status", res.Status),
			zap.String("latency", time.Since(start).String()),
			zap.String("id", id),
			zap.String("method", req.Method),
			zap.String("uri", req.RequestURI),
			zap.String("host", req.Host),
			zap.String("remote_ip", c.RealIP()),
		}

		n := res.Status
		switch {
		case n >= 500:
			m.logger.Error("Server error", fields...)
		case n >= 400:
			m.logger.Warn("Client error", fields...)
		case n >= 300:
			m.logger.Info("Redirection", fields...)
		default:
			m.logger.Info("Success", fields...)
		}

		return nil
	}
}

// HasRole validates that an authenticated user has at least one role from a
// specified list. This method constructs the actual function that is used.
func (m *GoMiddleware) HasRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*jwt.Token)
			claims, ok := user.Claims.(*auth.Claims)
			if !ok {
				return echo.NewHTTPError(http.StatusInternalServerError, "can't convert jwt.Claims to auth.Claims")
			}

			if !claims.HasRole(roles...) {
				return echo.NewHTTPError(http.StatusForbidden, "you are not authorized for that action")
			}

			return next(c)
		}
	}
}
