package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	urlHttp "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func newURL() *models.URL {
	return &models.URL{
		ID:             "test123",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}
}
func TestURLHttp_GetByID(t *testing.T) {
	tURL := newURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	t.Run("test get success", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, strings.NewReader(""))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id" + tURL.ID)
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.GetByID(c)
		assert.NoError(t, err)
		assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})

	t.Run("test get not found", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, models.ErrNotFound)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, strings.NewReader(""))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id" + tURL.ID)
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

	t.Run("test get success", func(t *testing.T) {
		uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		e := echo.New()
		req, err := http.NewRequest(echo.GET, "/"+tURL.ID, strings.NewReader(""))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:id" + tURL.ID)
		c.SetParamNames("id")
		c.SetParamValues(tURL.ID)
		handler := urlHttp.URLHandler{
			URLUsecase: uc,
		}

		err = handler.GetByID(c)
		assert.NoError(t, err)
		assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
		assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	})
}
