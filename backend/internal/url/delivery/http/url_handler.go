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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

// URLHandler represent the http handler for url
type URLHandler struct {
	urlUsecase    url.Usecase
	authenticator *auth.Authenticator
	validator     *web.AppValidator
	logger        *zap.Logger
	tracer        trace.Tracer
}

// NewURLHandler will initialize the url/ resources endpoint
func NewURLHandler(us url.Usecase, authenticator *auth.Authenticator, v *web.AppValidator, logger *zap.Logger, tracer trace.Tracer) (*URLHandler, error) {
	handler := &URLHandler{
		urlUsecase:    us,
		authenticator: authenticator,
		validator:     v,
		logger:        logger,
		tracer:        tracer,
	}

	err := handler.RegisterValidation()
	if err != nil {
		return nil, err
	}

	return handler, nil
}

// RegisterRoutes registers routes for a path with matching handler
func (uh *URLHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/v1/url/create", uh.Store)
	e.POST("/v1/user/url/create", uh.StoreUserURL, middleware.JWTWithConfig(uh.authenticator.JWTConfig))
	e.GET("/:id", uh.Redirect)
	e.GET("/v1/url/:id", uh.GetByID)
	e.DELETE("/v1/url/:id", uh.Delete, middleware.JWTWithConfig(uh.authenticator.JWTConfig))
	e.PUT("/v1/url", uh.Update, middleware.JWTWithConfig(uh.authenticator.JWTConfig))
}

// RegisterValidation will initialize validation for url handler
func (uh *URLHandler) RegisterValidation() error {
	err := uh.validator.V.RegisterValidation("linkid", checkURL)
	if err != nil {
		return err
	}

	err = uh.validator.V.RegisterTranslation("linkid", uh.validator.Translator, func(ut ut.Translator) error {
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

// Redirect will redirect to link by given id
func (uh *URLHandler) Redirect(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Redirect",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u, err := uh.getByID(ctx, c)
	if err != nil {
		span.RecordError(err)
		return err
	}

	if u != nil {
		span.SetStatus(codes.Ok, "success")
		return c.Redirect(http.StatusMovedPermanently, u.Link)
	}
	return nil
}

// GetByID will get url by given id
func (uh *URLHandler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http GetByID",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u, err := uh.getByID(ctx, c)
	if err != nil {
		span.RecordError(err)
		return err
	}

	if u != nil {
		span.SetStatus(codes.Ok, "success")
		return c.JSON(http.StatusOK, u)
	}
	return nil
}

func (uh *URLHandler) getByID(ctx context.Context, c echo.Context) (*models.URL, error) {
	id := c.Param("id")

	ctx, span := uh.tracer.Start(
		ctx,
		"http getByID",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	err := uh.validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		span.RecordError(err)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return nil, c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	u, err := uh.urlUsecase.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}
	span.SetAttributes(
		label.String("urlid", id),
	)

	return u, nil
}

// Store will store the URL by given request body
func (uh *URLHandler) Store(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Store",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u := new(models.CreateURL)
	return uh.storeURL(ctx, c, u)
}

// StoreUserURL will store the URL of authenticated user by given request body
func (uh *URLHandler) StoreUserURL(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http StoreUserURL",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u := new(models.CreateURL)
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		span.RecordError(web.ErrForbidden)
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		span.RecordError(web.ErrInternalServerError)
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	u.UserID = user.Subject

	span.SetAttributes(
		label.String("userid", user.Id),
	)

	return uh.storeURL(ctx, c, u)
}

func (uh *URLHandler) storeURL(ctx context.Context, c echo.Context, u *models.CreateURL) error {
	ctx, span := uh.tracer.Start(
		ctx,
		"http storeURL",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	if err := c.Bind(u); err != nil {
		span.RecordError(err)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(u); err != nil {
		span.RecordError(err)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	result, err := uh.urlUsecase.Store(ctx, *u)
	if err != nil {
		span.RecordError(err)
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	span.SetAttributes(
		label.String("urlid", result.ID),
	)

	return c.JSON(http.StatusCreated, result)
}

// Delete will delete URL by given id
func (uh *URLHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Delete",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	err := uh.validator.V.Var(id, "required,linkid,max=20")
	if err != nil {
		span.RecordError(err)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		span.RecordError(web.ErrForbidden)
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		span.RecordError(web.ErrInternalServerError)
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	if err = uh.urlUsecase.Delete(ctx, id, *user); err != nil {
		span.RecordError(err)
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	span.SetAttributes(
		label.String("userid", user.Id),
		label.String("urlid", id),
	)

	return c.JSON(http.StatusNoContent, nil)
}

// Update will update the URL by given request body
func (uh *URLHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := uh.tracer.Start(
		ctx,
		"http Update",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u := new(models.UpdateURL)
	if err := c.Bind(u); err != nil {
		span.RecordError(err)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: err.Error()})
	}

	if err := c.Validate(u); err != nil {
		span.RecordError(err)
		fields := err.(validator.ValidationErrors).Translate(uh.validator.Translator)
		return c.JSON(http.StatusBadRequest, web.ResponseError{Error: "validation error", Fields: fields})
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		span.RecordError(web.ErrForbidden)
		return c.JSON(http.StatusForbidden, web.ResponseError{Error: web.ErrForbidden.Error()})
	}
	user, ok := token.Claims.(*auth.Claims)
	if !ok {
		span.RecordError(web.ErrInternalServerError)
		return fmt.Errorf("%w can't convert jwt.Claims to auth.Claims", web.ErrInternalServerError)
	}

	if err := uh.urlUsecase.Update(ctx, *u, *user); err != nil {
		span.RecordError(err)
		return c.JSON(web.GetStatusCode(err, uh.logger), web.ResponseError{Error: err.Error()})
	}

	span.SetAttributes(
		label.String("userid", user.Id),
		label.String("urlid", u.ID),
	)

	return c.JSON(http.StatusNoContent, nil)
}
