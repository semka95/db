package http

import (
	"context"
	"net/http"
	"regexp"

	validator "github.com/go-playground/validator/v10"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"

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

// CreateID represent the response struct
type CreateID struct {
	ID string `json:"_id"`
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(e *echo.Echo, us url.Usecase) {
	handler := &URLHandler{
		URLUsecase: us,
	}
	e.POST("/url/create", handler.Store)
	e.GET("/:id", handler.GetByID)
	e.DELETE("/url/delete/:id", handler.Delete)
	e.PUT("/url/update/:id", handler.Update)
}

func checkURL(fl validator.FieldLevel) bool {
	r := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	return r.MatchString(fl.Field().String())
}

// GetByID will get url by given id
func (u *URLHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	validate := validator.New()
	err := validate.RegisterValidation("linkid", checkURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, nil)
	}

	err = validate.Var(id, "omitempty,linkid,min=7,max=20")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

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
	err := validate.RegisterValidation("linkid", checkURL)
	if err != nil {
		return false, err
	}

	err = validate.Struct(m)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Store will store the URL by given request body
func (u *URLHandler) Store(c echo.Context) error {
	var url models.URL
	if err := c.Bind(&url); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if ok, err := isRequestValid(&url); !ok {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := u.URLUsecase.Store(ctx, &url)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, CreateID{ID: id})
}

// Delete will delete URL by given id
func (u *URLHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	validate := validator.New()
	err := validate.RegisterValidation("linkid", checkURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	err = validate.Var(id, "omitempty,linkid,min=7,max=20")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err = u.URLUsecase.Delete(ctx, id); err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, nil)
}

// Update will update the URL by given request body
func (u *URLHandler) Update(c echo.Context) error {
	var url models.URL
	if err := c.Bind(&url); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if ok, err := isRequestValid(&url); !ok {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.URLUsecase.Update(ctx, &url); err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, url)
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
	case models.ErrNoAffected:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
