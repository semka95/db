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

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, "email must be a valid email address", body.Fields["CreateUser.email"])

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
		assert.Equal(t, http.StatusNoContent, rec.Code)
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

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, "email must be a valid email address", body.Fields["UpdateUser.email"])

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
		{"full_name greater than 30 symbols", "CreateUser.full_name", models.CreateUser{FullName: "qwertyuioasdfghjklzxcvbnmqwerta"}, "full_name must be a maximum of 30 characters in length"},
		{"email has wrong format", "CreateUser.email", models.CreateUser{Email: "wrong format"}, "email must be a valid email address"},
		{"email is empty", "CreateUser.email", models.CreateUser{Password: "test123456777"}, "email is a required field"},
		{"password less than 8 symbols", "CreateUser.password", models.CreateUser{Password: "sdf"}, "password must be at least 8 characters in length"},
		{"password greater than 30 symbols", "CreateUser.password", models.CreateUser{Password: "qwertyuuioppasdfghjklzxcvbnmmasdf"}, "password must be a maximum of 30 characters in length"},
		{"password is empty", "CreateUser.password", models.CreateUser{Email: "test@examle.com"}, "password is a required field"},
	}

	casesUpdateUser := []struct {
		Description string
		FieldName   string
		Data        models.UpdateUser
		Want        string
	}{
		{"id is empty", "UpdateUser.id", models.UpdateUser{Email: tests.StringPointer("test@examle.com")}, "id is a required field"},
		{"full_name greater than 30 symbols", "UpdateUser.full_name", models.UpdateUser{FullName: tests.StringPointer("qwertyuioasdfghjklzxcvbnmqwerta")}, "full_name must be a maximum of 30 characters in length"},
		{"email has wrong format", "UpdateUser.email", models.UpdateUser{Email: tests.StringPointer("wrong format")}, "email must be a valid email address"},
		{"password less than 8 symbols", "UpdateUser.password", models.UpdateUser{Password: tests.StringPointer("sdf")}, "password must be at least 8 characters in length"},
		{"password greater than 30 symbols", "UpdateUser.password", models.UpdateUser{Password: tests.StringPointer("qwertyuuioppasdfghjklzxcvbnmmasdf")}, "password must be a maximum of 30 characters in length"},
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
