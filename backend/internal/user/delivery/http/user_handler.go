package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	_MyMiddleware "bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
)

// UserHandler represent the http handler for user
type UserHandler struct {
	UserUsecase   user.Usecase
	Authenticator *auth.Authenticator
	Validator     *web.AppValidator
	Logger        *zap.Logger
}

// NewUserHandler will initialize the user/ resources endpoint
func NewUserHandler(e *echo.Echo, us user.Usecase, authenticator *auth.Authenticator, v *web.AppValidator, logger *zap.Logger) error {
	handler := &UserHandler{
		UserUsecase:   us,
		Authenticator: authenticator,
		Validator:     v,
		Logger:        logger,
	}

	myMiddl := _MyMiddleware.InitMiddleware(logger)

	e.POST("/v1/user/create", handler.Create)
	e.GET("/v1/user/:id", handler.GetByID, middleware.JWTWithConfig(authenticator.JWTConfig))
	e.GET("v1/user/token", handler.Token)
	e.DELETE("/v1/user/:id", handler.Delete, middleware.JWTWithConfig(authenticator.JWTConfig), myMiddl.HasRole(auth.RoleAdmin))
	e.PUT("/v1/user", handler.Update, middleware.JWTWithConfig(authenticator.JWTConfig))

	return nil
}

// GetByID will get user by given id
func (uh *UserHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	u, err := uh.UserUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, u)
}

// Create will store the User by given request body
func (uh *UserHandler) Create(c echo.Context) error {
	newUser := new(models.CreateUser)
	if err := c.Bind(newUser); err != nil {
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(newUser); err != nil {
		fields := err.(validator.ValidationErrors).Translate(uh.Validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	u, err := uh.UserUsecase.Create(ctx, *newUser)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, u)
}

// Delete will delete User by given id
func (uh *UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := uh.UserUsecase.Delete(ctx, id); err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Update will update the User by given request body
func (uh *UserHandler) Update(c echo.Context) error {
	u := new(models.UpdateUser)
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
	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := uh.UserUsecase.Update(ctx, *u, *claims); err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Token will return jwt token by given credentials
func (uh *UserHandler) Token(c echo.Context) error {
	email, pass, ok := c.Request().BasicAuth()
	if !ok {
		return c.JSON(http.StatusUnauthorized, web.ResponseError{Error: "can't get email and password using Basic auth"})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	claims, err := uh.UserUsecase.Authenticate(ctx, time.Now(), email, pass)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	var tkn struct {
		Token string `json:"token"`
	}
	tkn.Token, err = uh.Authenticator.GenerateToken(*claims)
	if err != nil {
		return c.JSON(web.GetStatusCode(err, uh.Logger), web.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, tkn)
}
