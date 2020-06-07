package http

import (
	"context"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

// URLHandler represent the httphandler for url
type URLHandler struct {
	URLUsecase url.Usecase
	Validator  *URLValidator
	logger     *zap.Logger
}

// URLValidator represent validation struct for url
type URLValidator struct {
	Uni   *ut.UniversalTranslator
	V     *validator.Validate
	Trans ut.Translator
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(e *echo.Echo, us url.Usecase, logger *zap.Logger) error {
	handler := &URLHandler{
		URLUsecase: us,
		Validator:  new(URLValidator),
		logger:     logger,
	}

	err := handler.InitValidation()
	if err != nil {
		return err
	}
	e.Validator = handler.Validator

	e.POST("/v1/url/create", handler.Store)
	e.GET("/:id", handler.GetByID)
	e.DELETE("/v1/url/:id", handler.Delete)
	e.PUT("/v1/url/", handler.Update)

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

	u.Validator.V.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return nil
}

func checkURL(fl validator.FieldLevel) bool {
	r := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	return r.MatchString(fl.Field().String())
}

// GetByID will get url by given id
func (u *URLHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	err := u.Validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	url, err := u.URLUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, u.logger), web.ResponseError{Error: err.Error()})
	}
	return c.Redirect(http.StatusMovedPermanently, url.Link)
}

// Store will store the URL by given request body
func (u *URLHandler) Store(c echo.Context) error {
	url := new(models.CreateURL)
	if err := c.Bind(url); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(url); err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	if user, ok := c.Get("user").(*jwt.Token); ok {
		claims := user.Claims.(auth.Claims)
		url.UserID = claims.Subject
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := u.URLUsecase.Store(ctx, url)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, u.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, result)
}

// Delete will delete URL by given id
func (u *URLHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	err := u.Validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err = u.URLUsecase.Delete(ctx, id); err != nil {
		return c.JSON(web.GetStatusCode(err, u.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Update will update the URL by given request body
func (u *URLHandler) Update(c echo.Context) error {
	url := new(models.UpdateURL)
	if err := c.Bind(url); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(url); err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user := token.Claims.(auth.Claims)

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.URLUsecase.Update(ctx, url, user); err != nil {
		return c.JSON(web.GetStatusCode(err, u.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}
