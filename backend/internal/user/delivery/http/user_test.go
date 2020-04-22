package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
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
	tUser := models.NewUser()

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
		assert.EqualValues(t, tUser, body)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("get user not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(nil, models.ErrNotFound)
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

		var body userHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, models.ErrNotFound.Error(), body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHttp_Create(t *testing.T) {
	tUser := models.NewUser()
	tUser.Password = "test12345"

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("create user success", func(t *testing.T) {
		uc.EXPECT().Create(gomock.Any(), tUser).Return(tUser.ID.Hex(), nil)
		e := echo.New()

		b, err := json.Marshal(tUser)
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

		var body userHttp.CreateID
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, tUser.ID.Hex(), body.ID)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("create user internal error", func(t *testing.T) {
		uc.EXPECT().Create(gomock.Any(), tUser).Return("", models.ErrInternalServerError)
		e := echo.New()

		b, err := json.Marshal(tUser)
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

		var body userHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, models.ErrInternalServerError.Error(), body.Message)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("create user validation error", func(t *testing.T) {
		e := echo.New()

		tUser.Email = "not an email"
		b, err := json.Marshal(tUser)
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

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "Email must be a valid email address", body["User.Email"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUserHttp_Delete(t *testing.T) {
	tUser := models.NewUser()

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
		uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(models.ErrNoAffected)
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

		var body userHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUserHttp_Update(t *testing.T) {
	tUser := models.NewUser()
	tUser.Password = "test12345"

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("update user success", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUser).Return(nil)
		e := echo.New()

		b, err := json.Marshal(tUser)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.PUT, "/user/"+tUser.ID.Hex(), bytes.NewBuffer(b))
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

		body := &models.User{}
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.EqualValues(t, tUser, body)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("update user not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUser).Return(models.ErrNoAffected)
		e := echo.New()

		b, err := json.Marshal(tUser)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/user/"+tUser.ID.Hex(), bytes.NewBuffer(b))
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

		var body userHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		require.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("update user validation error", func(t *testing.T) {
		e := echo.New()

		tUser.Email = "wrong format"
		b, err := json.Marshal(tUser)
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

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "Email must be a valid email address", body["User.Email"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestValidateUser(t *testing.T) {
	u := userHttp.UserHandler{
		Validator: new(userHttp.UserValidator),
	}
	err := u.InitValidation()
	require.NoError(t, err)

	cases := []struct {
		Description string
		FieldName   string
		Data        models.User
		Want        string
	}{
		{"FullName greater than 30 symbols", "User.FullName", models.User{FullName: "qwertyuioasdfghjklzxcvbnmqwerta"}, "FullName must be a maximum of 30 characters in length"},
		{"Email has wrong format", "User.Email", models.User{Email: "wrong format"}, "Email must be a valid email address"},
		{"Password less than 8 symbols", "User.Password", models.User{Password: "sdf"}, "Password must be at least 8 characters in length"},
		{"Password greater than 30 symbols", "User.Password", models.User{Password: "qwertyuuioppasdfghjklzxcvbnmmasdf"}, "Password must be a maximum of 30 characters in length"},
	}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}
}
