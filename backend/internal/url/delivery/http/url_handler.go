package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	validator "gopkg.in/go-playground/validator.v9"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

// ResponseError represent the reseponse error struct
type ResponseError struct {
	Message string `json:"message"`
}

// URLHandler represent the httphandler for url
type URLHandler struct {
	URLUsecase url.Usecase
}

type createID struct {
	ID string `json:"_id"`
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(e *echo.Echo, us url.Usecase) {
	handler := &URLHandler{
		URLUsecase: us,
	}
	e.POST("/create", handler.Store)
	e.GET("/*", handler.GetByID)
	e.DELETE("/*", handler.Delete)
	// e.PUT("/*", handler.Update)
}

// GetByID will get url by given id
func (u *URLHandler) GetByID(c echo.Context) error {
	id := c.Request().URL.Path[5:] // fix this

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	url, err := u.URLUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}
	return c.Redirect(http.StatusMovedPermanently, url.Link)
}

func isRequestValid(m *models.URL) (bool, error) {
	validate := validator.New()
	err := validate.Struct(m)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Store will store the URL by given request body
func (u *URLHandler) Store(c echo.Context) error {
	var url models.URL
	err := c.Bind(&url)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, err.Error())
	}

	if ok, err := isRequestValid(&url); !ok {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := u.URLUsecase.Store(ctx, &url)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, createID{ID: id})
}

// Delete will delete URL by given id
func (u *URLHandler) Delete(c echo.Context) error {
	id := c.Path()
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	err := u.URLUsecase.Delete(ctx, id)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func getStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	logrus.Error(err)
	switch err {
	case models.ErrInternalServerError:
		return http.StatusInternalServerError
	case models.ErrNotFound:
		return http.StatusNotFound
	case models.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
