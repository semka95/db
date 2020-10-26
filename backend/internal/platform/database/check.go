package database

import (
	"context"
	"net/http"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
)

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
		return c.JSON(http.StatusInternalServerError, web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}
