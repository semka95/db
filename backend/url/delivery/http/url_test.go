package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/tests"
	urlHttp "github.com/semka95/shortener/backend/url/delivery/http"
	"github.com/semka95/shortener/backend/url/mock"
	"github.com/semka95/shortener/backend/web"
	"github.com/semka95/shortener/backend/web/auth"
)

func TestURLHTTP(t *testing.T) {
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mock.NewMockURLUsecase(controller)

	tracer := sdktrace.NewTracerProvider().Tracer("")
	v, err := web.NewAppValidator()
	require.NoError(t, err)

	handler, err := urlHttp.NewURLHandler(uc, nil, v, nil, tracer)
	require.NoError(t, err)

	e := echo.New()
	req := new(http.Request)
	e.Validator = v
	c := e.NewContext(req, nil)

	// Test URLHandler.GetByID and Redirect
	tURL := tests.NewURL()

	casesGet := []struct {
		description   string
		mockCalls     func(muc *mock.MockURLUsecase)
		param         string
		handler       func(t *testing.T, c echo.Context)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Redirect success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
			},
			param: tURL.ID,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Redirect(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, tURL.Link, rec.Header().Get("Location"))
				assert.Equal(t, http.StatusMovedPermanently, rec.Code)
			},
		},
		{
			description: "Redirect not found",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, domain.ErrNotFound)
			},
			param: tURL.ID,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Redirect(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrNotFound.Error(), body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "Redirect validation error",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			param:       "te!t",
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Redirect(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "GetByID success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
			},
			param: tURL.ID,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.GetByID(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tURL, body)
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
	}

	for _, tc := range casesGet {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.GET, "/"+tc.param, nil)

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/:id")
			c.SetParamNames("id")
			c.SetParamValues(tc.param)

			tc.handler(t, c)

			tc.checkResponse(rec)
		})
	}

	// Test URLHandler.Store
	tCreateUserURL := tests.NewCreateURL()
	tCreateURL := tests.NewCreateURL()
	tCreateURL.UserID = ""
	tURLCr := tests.NewURL()
	tURLCr.UserID = ""
	tCreateURLBadID := tests.NewCreateURL()
	tCreateURLBadID.ID = tests.StringPointer("test!")

	createURLB, err := json.Marshal(tCreateURL)
	require.NoError(t, err)
	tCreateURLBadIDB, err := json.Marshal(tCreateURLBadID)
	require.NoError(t, err)
	createUserURLB, err := json.Marshal(tCreateUserURL)
	require.NoError(t, err)

	casesCreate := []struct {
		description   string
		mockCalls     func(muc *mock.MockURLUsecase)
		reqBody       *bytes.Buffer
		auth          bool
		handler       func(t *testing.T, c echo.Context)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "StoreUserURL success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateUserURL).Return(tURL, nil)
			},
			reqBody: bytes.NewBuffer(createUserURLB),
			auth:    true,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.StoreUserURL(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tURL, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "StoreUserURL jwt not set",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer(createUserURLB),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.StoreUserURL(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "Store success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(tURLCr, nil)
			},
			reqBody: bytes.NewBuffer(createURLB),
			auth:    false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tURLCr, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "Store already exists",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(nil, domain.ErrConflict)
			},
			reqBody: bytes.NewBuffer(createURLB),
			auth:    false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, domain.ErrConflict, body.Error)
				assert.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			description: "Store validation error",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer(tCreateURLBadIDB),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "id must contain only a-z, A-Z, 0-9, _, - characters", body.Fields["CreateURL.id"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "Store bad request data",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("bad data")),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Contains(t, body.Error, "Syntax error")
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}
	for _, tc := range casesCreate {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.POST, "/user/url/create", tc.reqBody)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/user/url/create")
			if tc.auth {
				c.Set("user", token)
			}

			tc.handler(t, c)

			tc.checkResponse(rec)
		})
	}

	// Test URLHandler.Delete
	tURLBadEmail := tests.NewURL()
	tURLBadEmail.ID = "te!t"

	casesDelete := []struct {
		description   string
		mockCalls     func(muc *mock.MockURLUsecase)
		auth          bool
		url           *domain.URL
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Delete success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(nil)
			},
			auth: true,
			url:  tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "Delete not authorized",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			auth:        false,
			url:         tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "Delete not existing url",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(domain.ErrNoAffected)
			},
			auth: true,
			url:  tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, domain.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "Delete validation error",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			auth:        true,
			url:         tURLBadEmail,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}
	for _, tc := range casesDelete {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.DELETE, "/"+tc.url.ID, nil)

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/delete/:id")
			c.SetParamNames("id")
			c.SetParamValues(tc.url.ID)
			if tc.auth {
				c.Set("user", token)
			}

			err = handler.Delete(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}

	// Test URLHandler.Update
	tUpdateURL := tests.NewUpdateURL()
	tUpdateURLBadID := tests.NewUpdateURL()
	tUpdateURLBadID.ID = ""

	tUpdateURLB, err := json.Marshal(tUpdateURL)
	require.NoError(t, err)
	tUpdateURLBadIDB, err := json.Marshal(tUpdateURLBadID)
	require.NoError(t, err)

	casesUpdate := []struct {
		description   string
		mockCalls     func(muc *mock.MockURLUsecase)
		reqBody       *bytes.Buffer
		token         *jwt.Token
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Update success",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(nil)
			},
			reqBody: bytes.NewBuffer(tUpdateURLB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "Update not authorized",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateURLB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "Update not exist",
			mockCalls: func(muc *mock.MockURLUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(domain.ErrNoAffected)
			},
			reqBody: bytes.NewBuffer(tUpdateURLB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, domain.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "Update validation error",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateURLBadIDB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "id is a required field", body.Fields["UpdateURL.id"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "Update bad request data",
			mockCalls:   func(muc *mock.MockURLUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("wrong data")),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Contains(t, body.Error, "Syntax error")
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range casesUpdate {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.PUT, "/url/update", tc.reqBody)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/url/update")
			c.Set("user", tc.token)

			err = handler.Update(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}

	// Test validation for models.CreateURL and models.UpdateURL structs
	casesCreateURL := []struct {
		description string
		fieldName   string
		data        domain.CreateURL
		want        string
	}{
		{
			description: "validate CreateURL id not valid format",
			fieldName:   "CreateURL.id",
			data:        domain.CreateURL{ID: tests.StringPointer("test1/,!")},
			want:        "id must contain only a-z, A-Z, 0-9, _, - characters",
		},
		{
			description: "validate CreateURL id too short",
			fieldName:   "CreateURL.id",
			data:        domain.CreateURL{ID: tests.StringPointer("tes")},
			want:        "id must be at least 7 characters in length",
		},
		{
			description: "validate CreateURL id too long",
			fieldName:   "CreateURL.id",
			data:        domain.CreateURL{ID: tests.StringPointer("testqwertyuiopasdfghj")},
			want:        "id must be a maximum of 20 characters in length",
		},
		{
			description: "validate CreateURL link not set",
			fieldName:   "CreateURL.link",
			data:        domain.CreateURL{ID: tests.StringPointer("test123")},
			want:        "link is a required field",
		},
		{
			description: "validate CreateURL link has wrong format",
			fieldName:   "CreateURL.link",
			data:        domain.CreateURL{ID: tests.StringPointer("test123"), Link: "not url"},
			want:        "link must be a valid URL",
		},
		{
			description: "validate CreateURL expiration date has wrong format",
			fieldName:   "CreateURL.expiration_date",
			data: domain.CreateURL{
				ID:             tests.StringPointer("test123"),
				Link:           "https://www.example.org",
				ExpirationDate: tests.DatePointer(time.Now().AddDate(0, 0, -1)),
			},
			want: "expiration_date must be greater than the current Date & Time",
		},
	}

	casesUpdateURL := []struct {
		description string
		fieldName   string
		data        domain.UpdateURL
		want        string
	}{
		{
			description: "validate UpdateURL id not set",
			fieldName:   "UpdateURL.id",
			want:        "id is a required field",
		},
		{
			description: "validate UpdateURL id not valid format",
			fieldName:   "UpdateURL.id",
			data:        domain.UpdateURL{ID: "test1/,!"},
			want:        "id must contain only a-z, A-Z, 0-9, _, - characters",
		},
		{
			description: "validate UpdateURL id too long",
			fieldName:   "UpdateURL.id",
			data:        domain.UpdateURL{ID: "testqwertyuiopasdfghj"},
			want:        "id must be a maximum of 20 characters in length",
		},
		{
			description: "validate UpdateURL expiration date not set",
			fieldName:   "UpdateURL.expiration_date",
			data:        domain.UpdateURL{ID: "test123"},
			want:        "expiration_date is a required field",
		},
		{
			description: "validate UpdateURL expiration date has wrong format",
			fieldName:   "UpdateURL.expiration_date",
			data: domain.UpdateURL{
				ID:             "test123",
				ExpirationDate: time.Now().AddDate(0, 0, -1),
			},
			want: "expiration_date must be greater than the current Date & Time"},
	}

	for _, tc := range casesCreateURL {
		t.Run(tc.description, func(t *testing.T) {
			if err := v.V.Struct(tc.data); err != nil {
				res := err.(validator.ValidationErrors).Translate(v.Translator)
				assert.Equal(t, tc.want, res[tc.fieldName])
			}
		})
	}

	for _, tc := range casesUpdateURL {
		t.Run(tc.description, func(t *testing.T) {
			if err := v.V.Struct(tc.data); err != nil {
				res := err.(validator.ValidationErrors).Translate(v.Translator)
				assert.Equal(t, tc.want, res[tc.fieldName])
			}
		})
	}
}
