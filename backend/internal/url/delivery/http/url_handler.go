package http

import (
	"context"
	"net/http"
	"regexp"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
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
	Validator  *URLValidator
}

// URLValidator represent validation struct for url
type URLValidator struct {
	Uni   *ut.UniversalTranslator
	V     *validator.Validate
	Trans ut.Translator
}

// CreateID represent the response struct
type CreateID struct {
	ID string `json:"_id"`
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(e *echo.Echo, us url.Usecase) error {
	handler := &URLHandler{
		URLUsecase: us,
		Validator:  new(URLValidator),
	}

	err := handler.InitValidation()
	if err != nil {
		return err
	}
	e.Validator = handler.Validator

	e.POST("/url/create", handler.Store)
	e.GET("/:id", handler.GetByID)
	e.DELETE("/url/delete/:id", handler.Delete)
	e.PUT("/url/update/:id", handler.Update)

	return nil
}

// Validate serving to be called by Echo to validate url
func (uv *URLValidator) Validate(i interface{}) error {
	return uv.V.Struct(i)
}

// InitValidation will initialize validation for url handler
func (u *URLHandler) InitValidation() error {
	en := en.New()
	u.Validator.Uni = ut.New(en, en)
	var found bool
	u.Validator.Trans, found = u.Validator.Uni.GetTranslator("en")
	if !found {
		u.Validator.Trans = u.Validator.Uni.GetFallback()
	}

	u.Validator.V = validator.New()
	err := u.Validator.V.RegisterValidation("linkid", checkURL)
	if err != nil {
		return err
	}

	err = u.Validator.V.RegisterTranslation("linkid", u.Validator.Trans, func(ut ut.Translator) error {
		return ut.Add("linkid", "{0} must contain only a-z, A-Z, 0-9, _, - characters", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("linkid", fe.Field())
		return t
	})
	if err != nil {
		return err
	}

	err = en_translations.RegisterDefaultTranslations(u.Validator.V, u.Validator.Trans)
	if err != nil {
		return err
	}

	return nil
}

func checkURL(fl validator.FieldLevel) bool {
	r := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	return r.MatchString(fl.Field().String())
}

// GetByID will get url by given id
func (u *URLHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	err := u.Validator.V.Var(id, "omitempty,linkid,min=7,max=20")
	if err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
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

// Store will store the URL by given request body
func (u *URLHandler) Store(c echo.Context) error {
	url := new(models.URL)
	if err := c.Bind(url); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := c.Validate(url); err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := u.URLUsecase.Store(ctx, url)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, CreateID{ID: id})
}

// Delete will delete URL by given id
func (u *URLHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	err := u.Validator.V.Var(id, "omitempty,linkid,min=7,max=20")
	if err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
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
	url := new(models.URL)
	if err := c.Bind(url); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := c.Validate(url); err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.URLUsecase.Update(ctx, url); err != nil {
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
