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
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
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
		}

		err := handler.GetByID(c)
		assert.NoError(t, err)
		assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})

	t.Run("test get not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, models.ErrNotFound)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, nil)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.GetByID(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, models.ErrNotFound.Error(), body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
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
		assert.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Store(c)
		assert.NoError(t, err)

		var body urlHttp.CreateID
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, tURL.ID, body.ID)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("test store bad request, empty body", func(t *testing.T) {
		e := echo.New()

		req, err := http.NewRequest(echo.POST, "/url/create", nil)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Store(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, models.ErrBadParamInput.Error(), body.Message)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("test store record already exists", func(t *testing.T) {
		uc.EXPECT().Store(gomock.Any(), tURL).Return("", models.ErrConflict)
		e := echo.New()

		b, err := json.Marshal(tURL)
		assert.NoError(t, err)
		req, err := http.NewRequest(echo.POST, "/url/create", bytes.NewBuffer(b))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/create")
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Store(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Error(t, models.ErrConflict, body.Message)

		assert.Equal(t, http.StatusConflict, rec.Code)
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
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("test delete not existed url", func(t *testing.T) {
		uc.EXPECT().Delete(gomock.Any(), tURL.ID).Return(models.ErrNoAffected)
		e := echo.New()
		req, err := http.NewRequest(echo.DELETE, "/delete/"+tURL.ID, nil)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/delete/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Delete(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
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
		assert.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, bytes.NewBuffer(b))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Update(c)
		assert.NoError(t, err)

		body := &models.URL{}
		err = json.NewDecoder(rec.Body).Decode(body)
		assert.NoError(t, err)
		assert.EqualValues(t, tURL, body)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("test error update empty request body", func(t *testing.T) {
		e := echo.New()

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, nil)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Update(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, models.ErrBadParamInput.Error(), body.Message)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("test update record not exist", func(t *testing.T) {
		uc.EXPECT().Update(gomock.Any(), tURL).Return(models.ErrNoAffected)
		e := echo.New()

		b, err := json.Marshal(tURL)
		assert.NoError(t, err)

		req, err := http.NewRequest(echo.PUT, "/url/update/"+tURL.ID, bytes.NewBuffer(b))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/url/update/:id")
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.Update(c)
		assert.NoError(t, err)

		var body urlHttp.ResponseError
		err = json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Error(t, models.ErrNoAffected, body.Message)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
