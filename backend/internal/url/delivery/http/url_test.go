package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	urlHttp "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	validator "github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLHttp_GetByID(t *testing.T) {
	tURL := models.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("test get success", func(t *testing.T) {
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

	t.Run("test get not found", func(t *testing.T) {
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

	t.Run("test get validation error", func(t *testing.T) {
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
	tURL := models.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("test store success", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tURL).Return(tURL.ID, nil)
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

		var body urlHttp.CreateID
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, tURL.ID, body.ID)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("test store record already exists", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tURL).Return("", models.ErrConflict)
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

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Error(t, models.ErrConflict, body.Message)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("test store validation error", func(t *testing.T) {
		e := echo.New()

		tURL.ID = "test!"
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

		var body map[string]string
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "ID must contain only a-z, A-Z, 0-9, _, - characters", body["URL.ID"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestURLHttp_Delete(t *testing.T) {
	tURL := models.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("test delete success", func(t *testing.T) {
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

	t.Run("test delete not existed url", func(t *testing.T) {
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
		assert.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("test delete validation error", func(t *testing.T) {
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
	tURL := models.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("test update success", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tURL).Return(nil)
		e := echo.New()

		b, err := json.Marshal(tURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		body := &models.URL{}
		err = json.NewDecoder(rec.Body).Decode(body)
		require.NoError(t, err)
		assert.EqualValues(t, tURL, body)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("test update record not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tURL).Return(models.ErrNoAffected)
		e := echo.New()

		b, err := json.Marshal(tURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

		handler := urlHttp.URLHandler{
			URLUsecase: uc,
			Validator:  new(urlHttp.URLValidator),
		}
		err = handler.InitValidation()
		c.Echo().Validator = handler.Validator
		require.NoError(t, err)

		err = handler.Update(c)
		require.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("test update validation error", func(t *testing.T) {
		e := echo.New()

		tURL.Link = "wrong format"
		b, err := json.Marshal(tURL)
		require.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, bytes.NewBuffer(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)

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
		assert.Equal(t, "Link must be a valid URL", body["URL.Link"])

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestValidateURL(t *testing.T) {
	u := urlHttp.URLHandler{
		Validator: new(urlHttp.URLValidator),
	}
	err := u.InitValidation()
	require.NoError(t, err)

	cases := []struct {
		Description string
		FieldName   string
		Data        models.URL
		Want        string
	}{
		{"test ID field not valid format", "URL.ID", models.URL{ID: "test1/,!"}, "ID must contain only a-z, A-Z, 0-9, _, - characters"},
		{"test ID field too short", "URL.ID", models.URL{ID: "tes"}, "ID must be at least 7 characters in length"},
		{"test ID field too long", "URL.ID", models.URL{ID: "testqwertyuiopasdfghj"}, "ID must be a maximum of 20 characters in length"},
		{"test Link field not set", "URL.Link", models.URL{ID: "test123"}, "Link is a required field"},
		{"test Link field field has wrong format", "URL.Link", models.URL{ID: "test123", Link: "not url"}, "Link must be a valid URL"},
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
