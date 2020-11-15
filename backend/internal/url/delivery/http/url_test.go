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

func TestURLHttp_RedirectAndGetByID(t *testing.T) {
	tURL := tests.NewURL()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := urlHttp.URLHandler{
		URLUsecase: uc,
		Validator:  v,
	}
	err = handler.RegisterValidation()
	require.NoError(t, err)

	e := echo.New()
	req := new(http.Request)
	c := e.NewContext(req, nil)

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		param         string
		handler       func(t *testing.T, c echo.Context)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Redirect success",
			mockCalls: func(muc *mocks.MockUsecase) {
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
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, web.ErrNotFound)
			},
			param: tURL.ID,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Redirect(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrNotFound.Error(), body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "Redirect validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			param:       "te!t",
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Redirect(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "GetByID success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
			},
			param: tURL.ID,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.GetByID(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(models.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tURL, body)
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
	}

	for _, tc := range cases {
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
}

func TestURLHttp_StoreUserURLandStore(t *testing.T) {
	tCreateUserURL := tests.NewCreateURL()
	tUserURL := tests.NewURL()
	tCreateURL := tests.NewCreateURL()
	tCreateURL.UserID = ""
	tURL := tests.NewURL()
	tURL.UserID = ""
	tCreateURLBadID := tests.NewCreateURL()
	tCreateURLBadID.ID = tests.StringPointer("test!")

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := urlHttp.URLHandler{
		URLUsecase: uc,
		Validator:  v,
	}
	err = handler.RegisterValidation()

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	createUserURLB, err := json.Marshal(tCreateUserURL)
	require.NoError(t, err)
	createURLB, err := json.Marshal(tCreateURL)
	require.NoError(t, err)
	tCreateURLBadIDB, err := json.Marshal(tCreateURLBadID)
	require.NoError(t, err)

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		reqBody       *bytes.Buffer
		auth          bool
		handler       func(t *testing.T, c echo.Context)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "StoreUserURL success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateUserURL).Return(tUserURL, nil)
			},
			reqBody: bytes.NewBuffer(createUserURLB),
			auth:    true,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.StoreUserURL(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(models.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tUserURL, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "StoreUserURL jwt not set",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(createUserURLB),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.StoreUserURL(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "Store success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(tURL, nil)
			},
			reqBody: bytes.NewBuffer(createURLB),
			auth:    false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(models.URL)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tURL, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "Store already exists",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Store(gomock.Any(), tCreateURL).Return(nil, web.ErrConflict)
			},
			reqBody: bytes.NewBuffer(createURLB),
			auth:    false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, web.ErrConflict, body.Error)
				assert.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			description: "Store validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tCreateURLBadIDB),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "id must contain only a-z, A-Z, 0-9, _, - characters", body.Fields["CreateURL.id"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "Store bad request data",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("bad data")),
			auth:        false,
			handler: func(t *testing.T, c echo.Context) {
				err = handler.Store(c)
				require.NoError(t, err)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Contains(t, body.Error, "Syntax error")
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}
	for _, tc := range cases {
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
}

func TestURLHttp_Delete(t *testing.T) {
	tURL := tests.NewURL()
	tURLBadEmail := tests.NewURL()
	tURLBadEmail.ID = "te!t"

	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := urlHttp.URLHandler{
		URLUsecase: uc,
		Validator:  v,
	}
	err = handler.RegisterValidation()

	e := echo.New()
	req := new(http.Request)
	c := e.NewContext(req, nil)
	e.Validator = v

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		auth          bool
		url           *models.URL
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(nil)
			},
			auth: true,
			url:  tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "not authorized",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			auth:        false,
			url:         tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "not existing url",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tURL.ID, claims).Return(web.ErrNoAffected)
			},
			auth: true,
			url:  tURL,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, web.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			auth:        true,
			url:         tURLBadEmail,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, " must contain only a-z, A-Z, 0-9, _, - characters", body.Fields[""])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}
	for _, tc := range cases {
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
}

func TestURLHttp_Update(t *testing.T) {
	tUpdateURL := tests.NewUpdateURL()
	tUpdateURLBadID := tests.NewUpdateURL()
	tUpdateURLBadID.ID = ""

	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := urlHttp.URLHandler{
		URLUsecase: uc,
		Validator:  v,
	}
	err = handler.RegisterValidation()

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	tUpdateURLB, err := json.Marshal(tUpdateURL)
	require.NoError(t, err)
	tUpdateURLBadIDB, err := json.Marshal(tUpdateURLBadID)
	require.NoError(t, err)

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		reqBody       *bytes.Buffer
		token         *jwt.Token
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(nil)
			},
			reqBody: bytes.NewBuffer(tUpdateURLB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "not authorized",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateURLB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrForbidden.Error(), body.Error)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			description: "not exist",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateURL, claims).Return(web.ErrNoAffected)
			},
			reqBody: bytes.NewBuffer(tUpdateURLB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				require.Error(t, web.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateURLBadIDB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "id is a required field", body.Fields["UpdateURL.id"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "bad request data",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("wrong data")),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Contains(t, body.Error, "Syntax error")
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range cases {
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
