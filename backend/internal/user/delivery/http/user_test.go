package http_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	userHttp "bitbucket.org/dbproject_ivt/db/backend/internal/user/delivery/http"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user/mocks"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/validator/v10"
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
	handler := userHttp.UserHandler{
		UserUsecase: uc,
	}

	e := echo.New()
	req := new(http.Request)
	c := e.NewContext(req, nil)
	var err error
	reqTarget := "/" + tUser.ID.Hex()

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(tUser, nil)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(models.User)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				tUser.HashedPassword = ""
				assert.EqualValues(t, tUser, body)
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			description: "not found",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(nil, web.ErrNotFound)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrNotFound.Error(), body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.GET, reqTarget, nil)

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/:id")
			c.SetParamNames("id")
			c.SetParamValues(tUser.ID.Hex())

			err = handler.GetByID(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}
}

func TestUserHttp_Create(t *testing.T) {
	tCreateUser := tests.NewCreateUser()
	tUser := tests.NewUser()
	tUser.HashedPassword = ""
	tCreateUserBadEmail := tests.NewCreateUser()
	tCreateUserBadEmail.Email = "bad email"

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := userHttp.UserHandler{
		UserUsecase: uc,
		Validator:   v,
		Logger:      zap.NewNop(),
	}

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	createUserB, err := json.Marshal(tCreateUser)
	require.NoError(t, err)
	tCreateUserBadEmailB, err := json.Marshal(tCreateUserBadEmail)
	require.NoError(t, err)

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		reqBody       *bytes.Buffer
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(tUser, nil)
			},
			reqBody: bytes.NewBuffer(createUserB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(models.User)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tUser, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "internal error",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(nil, web.ErrInternalServerError)
			},
			reqBody: bytes.NewBuffer(createUserB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, web.ErrInternalServerError.Error(), body.Error)
				assert.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			description: "validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tCreateUserBadEmailB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "email must be a valid email address", body.Fields["CreateUser.email"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "bad request data",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("wrong data")),
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
			req = httptest.NewRequest(echo.POST, "/user/create", tc.reqBody)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/user/create")

			err = handler.Create(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}
}

func TestUserHttp_Delete(t *testing.T) {
	tUser := tests.NewUser()

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)
	handler := userHttp.UserHandler{
		UserUsecase: uc,
	}

	e := echo.New()
	req := new(http.Request)
	c := e.NewContext(req, nil)
	var err error
	reqTarget := "/user/" + tUser.ID.Hex()

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(nil)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "existed user",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(web.ErrNoAffected)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Error(t, web.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.DELETE, reqTarget, nil)

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/user/:id")
			c.SetParamNames("id")
			c.SetParamValues(tUser.ID.Hex())

			err = handler.Delete(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}
}

func TestUserHttp_Update(t *testing.T) {
	tUpdateUser := tests.NewUpdateUser()
	tUpdateUserWrongEmail := tests.NewUpdateUser()
	tUpdateUserWrongEmail.Email = tests.StringPointer("wrong email")

	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := userHttp.UserHandler{
		UserUsecase: uc,
		Validator:   v,
	}

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	tUpdateUserB, err := json.Marshal(tUpdateUser)
	require.NoError(t, err)
	tUpdateUserWrongEmailB, err := json.Marshal(tUpdateUserWrongEmail)
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
				uc.EXPECT().Update(gomock.Any(), tUpdateUser, claims).Return(nil)
			},
			reqBody: bytes.NewBuffer(tUpdateUserB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "not authorized",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateUserB),
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
				uc.EXPECT().Update(gomock.Any(), tUpdateUser, claims).Return(web.ErrNoAffected)
			},
			reqBody: bytes.NewBuffer(tUpdateUserB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(&body)
				require.NoError(t, err)
				require.Error(t, web.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "validation error",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateUserWrongEmailB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "email must be a valid email address", body.Fields["UpdateUser.email"])
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
			req = httptest.NewRequest(echo.PUT, "/user/", tc.reqBody)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/user/")
			c.Set("user", tc.token)

			err = handler.Update(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}
}

func TestValidateUser(t *testing.T) {
	v, err := web.NewAppValidator()
	require.NoError(t, err)

	u := userHttp.UserHandler{
		Validator: v,
	}

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
		{"current_password is empty", "UpdateUser.current_password", models.UpdateUser{Email: tests.StringPointer("test@examle.com")}, "current_password is a required field"},
		{"current_password less than 8 symbols", "UpdateUser.current_password", models.UpdateUser{CurrentPassword: "sdf"}, "current_password must be at least 8 characters in length"},
		{"current_password greater than 30 symbols", "UpdateUser.current_password", models.UpdateUser{CurrentPassword: "qwertyuuioppasdfghjklzxcvbnmmasdf"}, "current_password must be a maximum of 30 characters in length"},
		{"new_password less than 8 symbols", "UpdateUser.new_password", models.UpdateUser{NewPassword: tests.StringPointer("sdf")}, "new_password must be at least 8 characters in length"},
		{"new_password greater than 30 symbols", "UpdateUser.new_password", models.UpdateUser{NewPassword: tests.StringPointer("qwertyuuioppasdfghjklzxcvbnmmasdf")}, "new_password must be a maximum of 30 characters in length"},
	}

	for _, test := range casesCreateUser {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Translator)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}

	for _, test := range casesUpdateUser {
		t.Run(test.Description, func(t *testing.T) {
			if err := u.Validator.V.Struct(test.Data); err != nil {
				res := err.(validator.ValidationErrors).Translate(u.Validator.Translator)
				assert.Equal(t, test.Want, res[test.FieldName])
			}
		})
	}
}

func TestUserHttp_Token(t *testing.T) {
	tUser := tests.NewUser()
	password := "password"
	claims := auth.NewClaims(tUser.ID.Hex(), tUser.Roles, time.Now(), time.Hour)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	kid := "4754d86b-7a6d-4df5-9c65-224741361492"
	kf := auth.NewSimpleKeyLookupFunc(kid, key.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(key, kid, "RS256", kf)
	require.NoError(t, err)

	token, err := authenticator.GenerateToken(claims)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mocks.NewMockUsecase(controller)

	v, err := web.NewAppValidator()
	require.NoError(t, err)
	handler := userHttp.UserHandler{
		UserUsecase:   uc,
		Authenticator: authenticator,
		Validator:     v,
	}

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	cases := []struct {
		description   string
		mockCalls     func(muc *mocks.MockUsecase)
		auth          bool
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "success",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), gomock.Any(), tUser.Email, password).Return(&claims, nil)
			},
			auth: true,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := make(map[string]string)
				err = json.NewDecoder(rec.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, token, body["token"])
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			description: "no credentials",
			mockCalls:   func(muc *mocks.MockUsecase) {},
			auth:        false,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "can't get email and password using Basic auth", body.Error)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			description: "authentication failure",
			mockCalls: func(muc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), gomock.Any(), tUser.Email, password).Return(nil, web.ErrAuthenticationFailure)
			},
			auth: true,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(web.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "authentication failed", body.Error)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.GET, "/user/token", nil)
			if tc.auth {
				req.SetBasicAuth(tUser.Email, password)
			}

			rec := httptest.NewRecorder()
			c.Reset(req, rec)
			c.SetPath("/user/token")

			err = handler.Token(c)
			require.NoError(t, err)

			tc.checkResponse(rec)
		})
	}
}
