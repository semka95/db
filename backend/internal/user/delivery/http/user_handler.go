package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
)

// ResponseError represent the reseponse error struct
type ResponseError struct {
	Message string `json:"message"`
}

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

// CreateID represent the response struct
type CreateID struct {
	ID string `json:"_id"`
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
	// err := u.Validator.V.RegisterValidation("linkid", checkURL)
	// if err != nil {
	// 	return err
	// }

	// err := u.Validator.V.RegisterTranslation("linkid", u.Validator.Trans, func(ut ut.Translator) error {
	// 	return ut.Add("linkid", "{0} must contain only a-z, A-Z, 0-9, _, - characters", true)
	// }, func(ut ut.Translator, fe validator.FieldError) string {
	// 	t, _ := ut.T("linkid", fe.Field())
	// 	return t
	// })
	// if err != nil {
	// 	return err
	// }

	err := en_translations.RegisterDefaultTranslations(u.Validator.V, u.Validator.Trans)
	if err != nil {
		return err
	}

	return nil
}

// func checkURL(fl validator.FieldLevel) bool {
// 	r := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
// 	return r.MatchString(fl.Field().String())
// }

// GetByID will get user by given id
func (u *UserHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	// err := u.Validator.V.Var(id, "omitempty,linkid,min=7,max=20")
	// if err != nil {
	// 	res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
	// 	return c.JSON(http.StatusBadRequest, res)
	// }

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	user, err := u.UserUsecase.GetByID(ctx, id)
	if err != nil {
		return c.JSON(u.getStatusCode(err), ResponseError{Message: err.Error()})
	}
	return c.JSON(http.StatusOK, user)
}

// Create will store the User by given request body
func (u *UserHandler) Create(c echo.Context) error {
	user := new(models.User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := c.Validate(user); err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := u.UserUsecase.Create(ctx, user)
	if err != nil {
		return c.JSON(u.getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, CreateID{ID: id})
}

// Delete will delete User by given id
func (u *UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	// err := u.Validator.V.Var(id, "omitempty,linkid,min=7,max=20")
	// if err != nil {
	// 	res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
	// 	return c.JSON(http.StatusBadRequest, res)
	// }

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.UserUsecase.Delete(ctx, id); err != nil {
		return c.JSON(u.getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, nil)
}

// Update will update the User by given request body
func (u *UserHandler) Update(c echo.Context) error {
	user := new(models.User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := c.Validate(user); err != nil {
		res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
		return c.JSON(http.StatusBadRequest, res)
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := u.UserUsecase.Update(ctx, user); err != nil {
		return c.JSON(u.getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

func (u *UserHandler) getStatusCode(err error) int {
	if errors.Is(err, models.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, models.ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, models.ErrNoAffected) {
		return http.StatusNotFound
	}

	u.Logger.Error("Server error: ", zap.Error(err))
	return http.StatusInternalServerError
}
