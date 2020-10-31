package http

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

// URLHandler represent the http handler for url
type URLHandler struct {
	URLUsecase    url.Usecase
	Authenticator *auth.Authenticator
	Validator     *web.AppValidator
	logger        *zap.Logger
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(e *echo.Echo, us url.Usecase, authenticator *auth.Authenticator, v *web.AppValidator, logger *zap.Logger) error {
	handler := &URLHandler{
		URLUsecase:    us,
		Authenticator: authenticator,
		Validator:     v,
		logger:        logger,
	}

	err := handler.RegisterValidation()
	if err != nil {
		return err
	}

	e.POST("/v1/url/create", handler.Store)
	e.POST("/v1/user/url/create", handler.StoreUserURL, middleware.JWTWithConfig(authenticator.JWTConfig))
	e.GET("/:id", handler.GetByID)
	e.DELETE("/v1/url/:id", handler.Delete, middleware.JWTWithConfig(authenticator.JWTConfig))
	e.PUT("/v1/url", handler.Update, middleware.JWTWithConfig(authenticator.JWTConfig))

	return nil
}

// RegisterValidation will initialize validation for url handler
func (uh *URLHandler) RegisterValidation() error {
	err := uh.Validator.V.RegisterValidation("linkid", checkURL)
	if err != nil {
		return err
	}

	err = uh.Validator.V.RegisterTranslation("linkid", uh.Validator.Translator, func(ut ut.Translator) error {
		return ut.Add("linkid", "{0} must contain only a-z, A-Z, 0-9, _, - characters", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("linkid", fe.Field())
		return t
	})
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
func (uh *URLHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	err := uh.Validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		fields := err.(validator.ValidationErrors).Translate(uh.Validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	u, err := uh.URLUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}
	return c.Redirect(http.StatusMovedPermanently, u.Link)
}

// Store will store the URL by given request body
func (uh *URLHandler) Store(c echo.Context) error {
	u := new(models.CreateURL)
	return uh.storeURL(c, u)
}

// StoreUserURL will store the URL of authenticated user by given request body
func (uh *URLHandler) StoreUserURL(c echo.Context) error {
	u := new(models.CreateURL)
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}
	u.UserID = user.Subject

	return uh.storeURL(c, u)
}

func (uh *URLHandler) storeURL(c echo.Context, u *models.CreateURL) error {
	if err := c.Bind(u); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(u); err != nil {
		fields := err.(validator.ValidationErrors).Translate(uh.Validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := uh.URLUsecase.Store(ctx, *u)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, result)
}

// Delete will delete URL by given id
func (uh *URLHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	err := uh.Validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		fields := err.(validator.ValidationErrors).Translate(uh.Validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err = uh.URLUsecase.Delete(ctx, id, *user); err != nil {
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Update will update the URL by given request body
func (uh *URLHandler) Update(c echo.Context) error {
	u := new(models.UpdateURL)
	if err := c.Bind(u); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(u); err != nil {
		fields := err.(validator.ValidationErrors).Translate(uh.Validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := uh.URLUsecase.Update(ctx, *u, *user); err != nil {
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}
