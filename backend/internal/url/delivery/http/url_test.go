package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	urlHttp "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLHttp_GetByID(t *testing.T) {
	tURL := tests.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)

	t.Run("get url success", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		e := echo.New()
		req := httptest.NewRequest(echo.GET, "/"+tURL.ID, nil)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err := handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)
		assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})

	t.Run("get url not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, web.ErrNotFound)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.Equal(t, web.ErrNotFound.Error(), body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get url validation error", func(t *testing.T) {
		e := echo.New()
		tURL.ID = "te!t"
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Store(t *testing.T) {
	tCreateURL := tests.NewCreateURL()
	tURL := tests.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	v, err := web.NewAppValidator()
	require.NoError(t, err)

	t.Run("store url success", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(tURL, nil)
		e := echo.New()

		b, err := json.Marshal(tCreateURL)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(models.URL)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.EqualValues(t, tURL, body)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("store url success by anon user", func(t *testing.T) {
		anonURL := tests.NewURL()
		anonURL.UserID = ""
		anonCreateURL := tests.NewCreateURL()
		anonCreateURL.UserID = ""

		uc.EXPECT().Store(gomock.Any(), anonCreateURL).Return(anonURL, nil)
		e := echo.New()

		b, err := json.Marshal(anonCreateURL)
		require.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(models.URL)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.EqualValues(t, anonURL, body)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("store url already exists", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(nil, web.ErrConflict)
		e := echo.New()

		b, err := json.Marshal(tURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		require.Error(t, web.ErrConflict, body.Error)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("store url validation error", func(t *testing.T) {
		e := echo.New()

		tCreateURL.ID = tests.StringPointer("test!")
		b, err := json.Marshal(tCreateURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, "id must contain only a-z, A-Z, 0-9, _, - characters", body.Fields["CreateURL.id"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("store url bad request data", func(t *testing.T) {
		e := echo.New()

		b := []byte("wrong data")

		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Contains(t, body.Error, "Syntax error")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Delete(t *testing.T) {
	tURL := tests.NewURL()
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)

	t.Run("delete url success", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(nil)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/delete/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("delete url not authorized", func(t *testing.T) {
		e := echo.New()

		req, err := http.NewRequest(echo.DELETE, "/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, web.ErrForbidden.Error(), body.Error)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("delete not existing url", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(web.ErrNoAffected)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/delete/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		require.Error(t, web.ErrNoAffected, body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("delete url validation error", func(t *testing.T) {
		e := echo.New()

		tURL.ID = "te!t"
		req, err := http.NewRequest(echo.DELETE, "/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Update(t *testing.T) {
	tUpdateURL := tests.NewUpdateURL()
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)

	t.Run("update url success", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(nil)
		e := echo.New()

		b, err := json.Marshal(tUpdateURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update")
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("update url not authorized", func(t *testing.T) {
		e := echo.New()

		b, err := json.Marshal(tUpdateURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, web.ErrForbidden.Error(), body.Error)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("update url not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(web.ErrNoAffected)
		e := echo.New()

		b, err := json.Marshal(tUpdateURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update")
		c.Set("user", token)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		require.Error(t, web.ErrNoAffected, body.Error)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("update url validation error", func(t *testing.T) {
		e := echo.New()

		tUpdateURL.ID = ""
		b, err := json.Marshal(tUpdateURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}
		err = handler.RegisterValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Equal(t, "validation error", body.Error)
		assert.Equal(t, "id is a required field", body.Fields["UpdateURL.id"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("update url bad request data", func(t *testing.T) {
		e := echo.New()

		b := []byte("wrong data")

		req, err := http.NewRequest(echo.PUT, "/url/update", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  v,
		}

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(web.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)

		assert.Contains(t, body.Error, "Syntax error")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestValidateURL(t *testing.T) {
	v, err := web.NewAppValidator()
	require.NoError(t, err)
	u := urlHttp.URLHandler{
		Validator: v,
	}
	err = u.RegisterValidation()
	require.NoError(t, err)

	casesCreateURL := []struct {
		Description string
		FieldName   string
		Data        models.CreateURL
		Want        string
	}{
		{"id not valid format", "CreateURL.id", models.CreateURL{ID: tests.StringPointer("test1/,!")}, "id must contain only a-z, A-Z, 0-9, _, - characters"},
		{"id too short", "CreateURL.id", models.CreateURL{ID: tests.StringPointer("tes")}, "id must be at least 7 characters in length"},
		{"id too long", "CreateURL.id", models.CreateURL{ID: tests.StringPointer("testqwertyuiopasdfghj")}, "id must be a maximum of 20 characters in length"},
		{"link not set", "CreateURL.link", models.CreateURL{ID: tests.StringPointer("test123")}, "link is a required field"},
		{"link has wrong format", "CreateURL.link", models.CreateURL{ID: tests.StringPointer("test123"), Link: "not url"}, "link must be a valid URL"},
		{"expiration date has wrong format", "CreateURL.expiration_date", models.CreateURL{ID: tests.StringPointer("test123"), Link: "https://www.example.org", ExpirationDate: time.Now().AddDate(0, 0, -1)}, "expiration_date must be greater than the current Date & Time"},
	}

	casesUpdateURL := []struct {
		Description string
		FieldName   string
		Data        models.UpdateURL
		Want        string
	}{
		{"id not set", "UpdateURL.id", models.UpdateURL{}, "id is a required field"},
		{"id not valid format", "UpdateURL.id", models.UpdateURL{ID: "test1/,!"}, "id must contain only a-z, A-Z, 0-9, _, - characters"},
		{"id too long", "UpdateURL.id", models.UpdateURL{ID: "testqwertyuiopasdfghj"}, "id must be a maximum of 20 characters in length"},
		{"expiration date not set", "UpdateURL.expiration_date", models.UpdateURL{ID: "test123"}, "expiration_date is a required field"},
		{"expiration date has wrong format", "UpdateURL.expiration_date", models.UpdateURL{ID: "test123", ExpirationDate: time.Now().AddDate(0, 0, -1)}, "expiration_date must be greater than the current Date & Time"},
	}

	for _, test := range casesCreateURL {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Translator)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}

	for _, test := range casesUpdateURL {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Translator)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}
}
