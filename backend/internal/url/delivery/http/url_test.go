package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	urlHttp "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	validator "github.com/go-playground/validator/v10"
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
			Validator:  new(urlHttp.URLValidator),
		}
		err := handler.InitValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)
		assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})

	t.Run("get url not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, models.ErrNotFound)
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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, models.ErrNotFound.Error(), body.Message)

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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.GetByID(c)
		require.NoError(t, err)

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body[""])
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Store(t *testing.T) {
	tCreateURL := tests.NewCreateURL()
	tURL := tests.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

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

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
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

	t.Run("store url already exists", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(nil, models.ErrConflict)
		e := echo.New()

		b, err := json.Marshal(tURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		body := new(urlHttp.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		require.Error(t, models.ErrConflict, body.Message)

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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Store(c)
		require.NoError(t, err)

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "ID must contain only a-z, A-Z, 0-9, _, - characters", body["CreateURL.ID"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Delete(t *testing.T) {
	tURL := tests.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("delete url success", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tURL.ID).Return(nil)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/delete/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("delete not existing url", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tURL.ID).Return(models.ErrNoAffected)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/delete/"+tURL.ID, nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		require.Error(t, models.ErrNoAffected, body.Message)

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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		require.NoError(t, err)

		err = handler.Delete(c)
		require.NoError(t, err)

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body[""])
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Update(t *testing.T) {
	tUpdateURL := tests.NewUpdateURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("update url success", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateURL).Return(nil)
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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("update url not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tUpdateURL).Return(models.ErrNoAffected)
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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := new(urlHttp.ResponseError)
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		require.Error(t, models.ErrNoAffected, body.Message)

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
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "ID is a required field", body["UpdateURL.ID"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestValidateURL(t *testing.T) {
	u := urlHttp.URLHandler{
		Validator: new(urlHttp.URLValidator),
	}
	err := u.InitValidation()
	require.NoError(t, err)

	casesCreateURL := []struct {
		Description string
		FieldName   string
		Data        models.CreateURL
		Want        string
	}{
		{"ID not valid format", "CreateURL.ID", models.CreateURL{ID: tests.StringPointer("test1/,!")}, "ID must contain only a-z, A-Z, 0-9, _, - characters"},
		{"ID too short", "CreateURL.ID", models.CreateURL{ID: tests.StringPointer("tes")}, "ID must be at least 7 characters in length"},
		{"ID too long", "CreateURL.ID", models.CreateURL{ID: tests.StringPointer("testqwertyuiopasdfghj")}, "ID must be a maximum of 20 characters in length"},
		{"Link not set", "CreateURL.Link", models.CreateURL{ID: tests.StringPointer("test123")}, "Link is a required field"},
		{"Link has wrong format", "CreateURL.Link", models.CreateURL{ID: tests.StringPointer("test123"), Link: "not url"}, "Link must be a valid URL"},
		{"Expiration date has wrong format", "CreateURL.ExpirationDate", models.CreateURL{ID: tests.StringPointer("test123"), Link: "https://www.example.org", ExpirationDate: time.Now().AddDate(0, 0, -1)}, "ExpirationDate must be greater than the current Date & Time"},
	}

	casesUpdateURL := []struct {
		Description string
		FieldName   string
		Data        models.UpdateURL
		Want        string
	}{
		{"ID not set", "UpdateURL.ID", models.UpdateURL{}, "ID is a required field"},
		{"ID not valid format", "UpdateURL.ID", models.UpdateURL{ID: "test1/,!"}, "ID must contain only a-z, A-Z, 0-9, _, - characters"},
		{"ID too long", "UpdateURL.ID", models.UpdateURL{ID: "testqwertyuiopasdfghj"}, "ID must be a maximum of 20 characters in length"},
		{"Expiration date not set", "UpdateURL.ExpirationDate", models.UpdateURL{ID: "test123"}, "ExpirationDate is a required field"},
		{"Expiration date has wrong format", "UpdateURL.ExpirationDate", models.UpdateURL{ID: "test123", ExpirationDate: time.Now().AddDate(0, 0, -1)}, "ExpirationDate must be greater than the current Date & Time"},
	}

	for _, test := range casesCreateURL {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}

	for _, test := range casesUpdateURL {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Trans)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}
}
