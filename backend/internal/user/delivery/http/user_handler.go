package http

import (
	"context"
	"net/http"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
)

// UserHandler represent the httphandler for user
type UserHandler struct {
	UserUsecase user.Usecase
	Validator   *UserValidator
	Logger      *zap.Logger
}

// UserValidator represent validation struct for user
type UserValidator struct {
	Uni   *ut.UniversalTranslator
	V     *validator.Validate
	Trans ut.Translator
}

// NewUserHandler will initialize the user/ resources endpoint
func NewUserHandler(e *echo.Echo, us user.Usecase, logger *zap.Logger) error {
	handler := &UserHandler{
		UserUsecase: us,
		Validator:   new(UserValidator),
		Logger:      logger,
	}

	err := handler.InitValidation()
	if err != nil {
		return err
	}
	e.Validator = handler.Validator

	e.POST("/v1/user/create", handler.Create)
	e.GET("/v1/user/:id", handler.GetByID)
	e.DELETE("/v1/user/:id", handler.Delete)
	e.PUT("/v1/user/", handler.Update)

	return nil
}

// Validate serving to be called by Echo to validate user
func (uv *UserValidator) Validate(i interface{}) error {
	return uv.V.Struct(i)
}

// InitValidation will initialize validation for user handler
func (u *UserHandler) InitValidation() error {
	en := en.New()
	u.Validator.Uni = ut.New(en, en)
	var found bool
	u.Validator.Trans, found = u.Validator.Uni.GetTranslator("en")
	if !found {
		u.Validator.Trans = u.Validator.Uni.GetFallback()
	}

	u.Validator.V = validator.New()

	err := en_translations.RegisterDefaultTranslations(u.Validator.V, u.Validator.Trans)
	if err != nil {
		return err
	}

	return nil
}

// GetByID will get user by given id
func (u *UserHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	user, err := u.UserUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, u.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

// Create will store the User by given request body
func (u *UserHandler) Create(c echo.Context) error {
	newUser := new(models.CreateUser)
	if err := c.Bind(newUser); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(newUser); err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "Validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	user, err := u.UserUsecase.Create(ctx, newUser)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, u.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, user)
}

// Delete will delete User by given id
func (u *UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.UserUsecase.Delete(ctx, id); err != nil {
		return c.JSON(web.GetStatusCode(err, u.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, nil)
}

// Update will update the User by given request body
func (u *UserHandler) Update(c echo.Context) error {
	user := new(models.UpdateUser)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(user); err != nil {
		fields := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "Validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.UserUsecase.Update(ctx, user); err != nil {
		return c.JSON(web.GetStatusCode(err, u.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}
