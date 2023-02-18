package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/semka95/shortener/backend/domain"
	_MyMiddleware "github.com/semka95/shortener/backend/middleware"
	"github.com/semka95/shortener/backend/web"
	"github.com/semka95/shortener/backend/web/auth"
)

// UserHandler represent the http handler for user
type UserHandler struct {
	userUsecase   domain.UserUsecase
	authenticator *auth.Authenticator
	validator     *web.AppValidator
	logger        *zap.Logger
	tracer        trace.Tracer
}

// NewUserHandler will initialize the user/ resources endpoint
func NewUserHandler(us domain.UserUsecase, authenticator *auth.Authenticator, v *web.AppValidator, logger *zap.Logger, tracer trace.Tracer) *UserHandler {
	return &UserHandler{
		userUsecase:   us,
		authenticator: authenticator,
		validator:     v,
		logger:        logger,
		tracer:        tracer,
	}
}

// RegisterRoutes registers routes for a path with matching handler
func (uh *UserHandler) RegisterRoutes(e *echo.Echo) {
	myMiddl := _MyMiddleware.InitMiddleware(uh.logger)
	e.POST("/v1/user/create", uh.Create)
	e.GET("/v1/user/:id", uh.GetByID, echojwt.WithConfig(uh.authenticator.JWTConfig))
	e.GET("v1/user/token", uh.Token)
	e.DELETE("/v1/user/:id", uh.Delete, echojwt.WithConfig(uh.authenticator.JWTConfig), myMiddl.HasRole(auth.RoleAdmin))
	e.PUT("/v1/user", uh.Update, echojwt.WithConfig(uh.authenticator.JWTConfig))
}

// GetByID will get user by given id
func (uh *UserHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http GetByID",
	)
	defer span.End()

	u, err := uh.userUsecase.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}
	span.SetAttributes(
		attribute.String("userid", u.ID.Hex()),
	)

	return c.JSON(http.StatusOK, u)
}

// Create will store the User by given request body
func (uh *UserHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Create",
	)
	defer span.End()

	newUser := new(domain.CreateUser)
	if err := c.Bind(newUser); err != nil {
		span.RecordError(domain.ErrForbidden)
		return c.JSON(http.StatusBadRequest, domain.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(newUser); err != nil {
		span.RecordError(domain.ErrForbidden)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return c.JSON(http.StatusBadRequest, domain.ResponseError{Error: "validation error", Fields: fields})
	}

	u, err := uh.userUsecase.Create(ctx, *newUser)
	if err != nil {
		span.RecordError(domain.ErrForbidden)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}
	span.SetAttributes(
		attribute.String("userid", u.ID.Hex()),
	)

	return c.JSON(http.StatusCreated, u)
}

// Delete will delete User by given id
func (uh *UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Delete",
	)
	defer span.End()

	if err := uh.userUsecase.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Update will update the User by given request body
func (uh *UserHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Update",
	)
	defer span.End()

	u := new(domain.UpdateUser)
	if err := c.Bind(u); err != nil {
		span.RecordError(err)
		return c.JSON(http.StatusBadRequest, domain.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(u); err != nil {
		span.RecordError(err)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return c.JSON(http.StatusBadRequest, domain.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		span.RecordError(domain.ErrForbidden)
		return c.JSON(http.StatusForbidden, domain.ResponseError{Error: domain.ErrForbidden.Error()})
	}
	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		span.RecordError(domain.ErrInternalServerError)
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", domain.ErrInternalServerError)
	}

	if err := uh.userUsecase.Update(ctx, *u, claims); err != nil {
		span.RecordError(err)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Token will return jwt token by given credentials
func (uh *UserHandler) Token(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Token",
	)
	defer span.End()

	email, pass, ok := c.Request().BasicAuth()
	if !ok {
		span.RecordError(domain.ErrBadParamInput)
		return c.JSON(http.StatusUnauthorized, domain.ResponseError{Error: "can't get email and password using Basic auth"})
	}

	claims, err := uh.userUsecase.Authenticate(ctx, time.Now(), email, pass)
	if err != nil {
		span.RecordError(err)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}

	var tkn struct {
		Token string `json:"token"`
	}
	tkn.Token, err = uh.authenticator.GenerateToken(claims)
	if err != nil {
		span.RecordError(err)
		return c.JSON(domain.GetStatusCode(err, uh.logger), domain.ResponseError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, tkn)
}
