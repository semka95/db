package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	userHttp "bitbucket.org/dbproject_ivt/db/backend/internal/user/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user/mocks"
	validator "github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUserHttp_GetByID(t *testing.T) {
	tUser := tests.NewUser()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("get user success", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(tUser, nil)
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/"+tUser.ID.Hex(), nil)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tUser.ID.Hex())

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err := handler.InitValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		body := new(models.User)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		tUser.HashedPassword = ""
		assert.EqualValues(t, tUser, body)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("get user not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(nil, web.ErrNotFound)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tUser.ID.Hex(), nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tUser.ID.Hex())

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.Equal(t, web.ErrNotFound.Error(), body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHttp_Create(t *testing.T) {
	tCreateUser := tests.NewCreateUser()
	tUser := tests.NewUser()
	tUser.HashedPassword = ""

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("create user success", func(t *testing.T) {
		uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(tUser, nil)
		e := echo.New()

		b, err := json.Marshal(tCreateUser)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/user/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/create")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Create(c)
		require.NoError(t, err)

		body := new(models.User)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.EqualValues(t, tUser, body)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("create user internal error", func(t *testing.T) {
		uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(nil, web.ErrInternalServerError)
		e := echo.New()

		b, err := json.Marshal(tCreateUser)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/user/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/create")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
			Logger:      zap.NewExample(),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Create(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.Equal(t, web.ErrInternalServerError.Error(), body.Error)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("create user validation error", func(t *testing.T) {
		e := echo.New()

		tCreateUser.Email = "not an email"
		b, err := json.Marshal(tCreateUser)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.POST, "/user/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/create")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Create(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "Validation error", body.Error)
		assert.Equal(t, "Email must be a valid email address", body.Fields["CreateUser.Email"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUserHttp_Delete(t *testing.T) {
	tUser := tests.NewUser()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("delete success", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(nil)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/user/"+tUser.ID.Hex(), nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/:id")
		c.SetParamNames("id")
		c.SetParamValues(tUser.ID.Hex())

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("delete not existed user", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(web.ErrNoAffected)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/user/"+tUser.ID.Hex(), nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/:id")
		c.SetParamNames("id")
		c.SetParamValues(tUser.ID.Hex())

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.Error(t, web.ErrNoAffected, body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHttp_Update(t *testing.T) {
	tUpdateUser := tests.NewUpdateUser()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("update user success", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateUser).Return(nil)
		e := echo.New()

		b, err := json.Marshal(tUpdateUser)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.PUT, "/user/", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("update user not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateUser).Return(web.ErrNoAffected)
		e := echo.New()

		b, err := json.Marshal(tUpdateUser)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/user/", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		require.Error(t, web.ErrNoAffected, body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("update user validation error", func(t *testing.T) {
		e := echo.New()

		tUpdateUser.Email = tests.StringPointer("wrong format")
		b, err := json.Marshal(tUpdateUser)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/user/", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/user/")

		handler := userHttp.UserHandler{
			UserUsecase: uc,
			Validator:   new(userHttp.UserValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "Validation error", body.Error)
		assert.Equal(t, "Email must be a valid email address", body.Fields["UpdateUser.Email"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestValidateUser(t *testing.T) {
	u := userHttp.UserHandler{
		Validator: new(userHttp.UserValidator),
	}
	err := u.InitValidation()
	require.NoError(t, err)

	casesCreateUser := []struct {
		Description string
		FieldName   string
		Data        models.CreateUser
		Want        string
	}{
		{"FullName greater than 30 symbols", "CreateUser.FullName", models.CreateUser{FullName: "qwertyuioasdfghjklzxcvbnmqwerta"}, "FullName must be a maximum of 30 characters in length"},
		{"Email has wrong format", "CreateUser.Email", models.CreateUser{Email: "wrong format"}, "Email must be a valid email address"},
		{"Email is empty", "CreateUser.Email", models.CreateUser{Password: "test123456777"}, "Email is a required field"},
		{"Password less than 8 symbols", "CreateUser.Password", models.CreateUser{Password: "sdf"}, "Password must be at least 8 characters in length"},
		{"Password greater than 30 symbols", "CreateUser.Password", models.CreateUser{Password: "qwertyuuioppasdfghjklzxcvbnmmasdf"}, "Password must be a maximum of 30 characters in length"},
		{"Password is empty", "CreateUser.Password", models.CreateUser{Email: "test@examle.com"}, "Password is a required field"},
	}

	casesUpdateUser := []struct {
		Description string
		FieldName   string
		Data        models.UpdateUser
		Want        string
	}{
		{"ID is empty", "UpdateUser.ID", models.UpdateUser{Email: tests.StringPointer("test@examle.com")}, "ID is a required field"},
		{"FullName greater than 30 symbols", "UpdateUser.FullName", models.UpdateUser{FullName: tests.StringPointer("qwertyuioasdfghjklzxcvbnmqwerta")}, "FullName must be a maximum of 30 characters in length"},
		{"Email has wrong format", "UpdateUser.Email", models.UpdateUser{Email: tests.StringPointer("wrong format")}, "Email must be a valid email address"},
		{"Password less than 8 symbols", "UpdateUser.Password", models.UpdateUser{Password: tests.StringPointer("sdf")}, "Password must be at least 8 characters in length"},
		{"Password greater than 30 symbols", "UpdateUser.Password", models.UpdateUser{Password: tests.StringPointer("qwertyuuioppasdfghjklzxcvbnmmasdf")}, "Password must be a maximum of 30 characters in length"},
	}

	for _, test := range casesCreateUser {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}

	for _, test := range casesUpdateUser {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}
}
