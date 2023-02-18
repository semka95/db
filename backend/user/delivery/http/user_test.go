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

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/tests"
	userHttp "github.com/semka95/shortener/backend/user/delivery/http"
	"github.com/semka95/shortener/backend/user/mock"
	"github.com/semka95/shortener/backend/web"
	"github.com/semka95/shortener/backend/web/auth"
)

func TestUserHTTP(t *testing.T) {
	tUser := tests.NewUser()
	password := "password"
	claims := auth.NewClaims(tUser.ID.Hex(), tUser.Roles, time.Now(), time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	kid := "4754d86b-7a6d-4df5-9c65-224741361492"
	kf := auth.NewSimpleKeyLookupFunc(kid, key.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(key, kid, "RS256", kf)
	require.NoError(t, err)

	tokenStr, err := authenticator.GenerateToken(claims)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()
	uc := mock.NewMockUserUsecase(controller)

	tracer := sdktrace.NewTracerProvider().Tracer("")
	v, err := web.NewAppValidator()
	require.NoError(t, err)

	handler := userHttp.NewUserHandler(uc, authenticator, v, zap.NewNop(), tracer)

	e := echo.New()
	e.Validator = v
	req := new(http.Request)
	c := e.NewContext(req, nil)

	// Test UserHandler.GetByID
	reqTarget := "/" + tUser.ID.Hex()

	casesGet := []struct {
		description   string
		mockCalls     func(muc *mock.MockUserUsecase)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "GetByID success",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(tUser, nil)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.User)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				tUser.HashedPassword = ""
				assert.EqualValues(t, tUser, body)
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			description: "GetByID not found",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().GetByID(gomock.Any(), tUser.ID.Hex()).Return(nil, domain.ErrNotFound)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrNotFound.Error(), body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range casesGet {
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

	// Test UserHandler.Create
	tCreateUser := tests.NewCreateUser()
	tUserCr := tests.NewUser()
	tUserCr.HashedPassword = ""
	tCreateUserBadEmail := tests.NewCreateUser()
	tCreateUserBadEmail.Email = "bad email"

	createUserB, err := json.Marshal(tCreateUser)
	require.NoError(t, err)
	tCreateUserBadEmailB, err := json.Marshal(tCreateUserBadEmail)
	require.NoError(t, err)

	casesCreate := []struct {
		description   string
		mockCalls     func(muc *mock.MockUserUsecase)
		reqBody       *bytes.Buffer
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Create success",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(tUserCr, nil)
			},
			reqBody: bytes.NewBuffer(createUserB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.User)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.EqualValues(t, tUserCr, body)
				assert.Equal(t, http.StatusCreated, rec.Code)
			},
		},
		{
			description: "Create internal error",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Create(gomock.Any(), tCreateUser).Return(nil, domain.ErrInternalServerError)
			},
			reqBody: bytes.NewBuffer(createUserB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, domain.ErrInternalServerError.Error(), body.Error)
				assert.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			description: "Create validation error",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
			reqBody:     bytes.NewBuffer(tCreateUserBadEmailB),
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "email must be a valid email address", body.Fields["CreateUser.email"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "Create bad request data",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
			reqBody:     bytes.NewBuffer([]byte("wrong data")),
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

	// Test UserHandler.Delete
	casesDelete := []struct {
		description   string
		mockCalls     func(muc *mock.MockUserUsecase)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Delete success",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(nil)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "Delete existed user",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Delete(gomock.Any(), tUser.ID.Hex()).Return(domain.ErrNoAffected)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Error(t, domain.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range casesDelete {
		t.Run(tc.description, func(t *testing.T) {
			tc.mockCalls(uc)
			req = httptest.NewRequest(echo.DELETE, "/user/"+tUser.ID.Hex(), nil)

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

	// Test UserHandler.Update
	tUpdateUser := tests.NewUpdateUser()
	tUpdateUserWrongEmail := tests.NewUpdateUser()
	tUpdateUserWrongEmail.Email = tests.StringPointer("wrong email")

	tUpdateUserB, err := json.Marshal(tUpdateUser)
	require.NoError(t, err)
	tUpdateUserWrongEmailB, err := json.Marshal(tUpdateUserWrongEmail)
	require.NoError(t, err)

	casesUpdate := []struct {
		description   string
		mockCalls     func(muc *mock.MockUserUsecase)
		reqBody       *bytes.Buffer
		token         *jwt.Token
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Update success",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateUser, claims).Return(nil)
			},
			reqBody: bytes.NewBuffer(tUpdateUserB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			description: "Update not authorized",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateUserB),
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
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Update(gomock.Any(), tUpdateUser, claims).Return(domain.ErrNoAffected)
			},
			reqBody: bytes.NewBuffer(tUpdateUserB),
			token:   token,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(&body)
				require.NoError(t, err)
				require.Error(t, domain.ErrNoAffected, body.Error)
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			description: "Update validation error",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
			reqBody:     bytes.NewBuffer(tUpdateUserWrongEmailB),
			token:       nil,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "validation error", body.Error)
				assert.Equal(t, "email must be a valid email address", body.Fields["UpdateUser.email"])
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			description: "Update bad request data",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
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

	// Test UserHandler.Authenticate
	casesAuth := []struct {
		description   string
		mockCalls     func(muc *mock.MockUserUsecase)
		auth          bool
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			description: "Token success",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), gomock.Any(), tUser.Email, password).Return(claims, nil)
			},
			auth: true,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := make(map[string]string)
				err = json.NewDecoder(rec.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, tokenStr, body["token"])
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			description: "Token no credentials",
			mockCalls:   func(muc *mock.MockUserUsecase) {},
			auth:        false,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "can't get email and password using Basic auth", body.Error)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			description: "Token authentication failure",
			mockCalls: func(muc *mock.MockUserUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), gomock.Any(), tUser.Email, password).Return(nil, domain.ErrAuthenticationFailure)
			},
			auth: true,
			checkResponse: func(rec *httptest.ResponseRecorder) {
				body := new(domain.ResponseError)
				err = json.NewDecoder(rec.Body).Decode(body)
				require.NoError(t, err)
				assert.Equal(t, "authentication failed", body.Error)
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range casesAuth {
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

	// Test validation for models.CreateUser and models.UpdateUser structs
	casesCreateUser := []struct {
		description string
		fieldName   string
		data        domain.CreateUser
		want        string
	}{
		{
			description: "validate full_name greater than 30 symbols",
			fieldName:   "CreateUser.full_name",
			data:        domain.CreateUser{FullName: "qwertyuioasdfghjklzxcvbnmqwerta"},
			want:        "full_name must be a maximum of 30 characters in length",
		},
		{
			description: "validate email has wrong format",
			fieldName:   "CreateUser.email",
			data:        domain.CreateUser{Email: "wrong format"},
			want:        "email must be a valid email address",
		},
		{
			description: "validate email is empty",
			fieldName:   "CreateUser.email",
			data:        domain.CreateUser{Password: "test123456777"},
			want:        "email is a required field",
		},
		{
			description: "validate password less than 8 symbols",
			fieldName:   "CreateUser.password",
			data:        domain.CreateUser{Password: "sdf"},
			want:        "password must be at least 8 characters in length",
		},
		{
			description: "validate password greater than 30 symbols",
			fieldName:   "CreateUser.password",
			data:        domain.CreateUser{Password: "qwertyuuioppasdfghjklzxcvbnmmasdf"},
			want:        "password must be a maximum of 30 characters in length",
		},
		{
			description: "validate password is empty",
			fieldName:   "CreateUser.password",
			data:        domain.CreateUser{Email: "test@examle.com"},
			want:        "password is a required field",
		},
	}

	casesUpdateUser := []struct {
		description string
		fieldName   string
		data        domain.UpdateUser
		want        string
	}{
		{
			description: "validate id is empty",
			fieldName:   "UpdateUser.id",
			data:        domain.UpdateUser{Email: tests.StringPointer("test@examle.com")},
			want:        "id is a required field",
		},
		{
			description: "validate full_name greater than 30 symbols",
			fieldName:   "UpdateUser.full_name",
			data:        domain.UpdateUser{FullName: tests.StringPointer("qwertyuioasdfghjklzxcvbnmqwerta")},
			want:        "full_name must be a maximum of 30 characters in length",
		},
		{
			description: "validate email has wrong format",
			fieldName:   "UpdateUser.email",
			data:        domain.UpdateUser{Email: tests.StringPointer("wrong format")},
			want:        "email must be a valid email address",
		},
		{
			description: "validate current_password is empty",
			fieldName:   "UpdateUser.current_password",
			data:        domain.UpdateUser{Email: tests.StringPointer("test@examle.com")},
			want:        "current_password is a required field",
		},
		{
			description: "validate current_password less than 8 symbols",
			fieldName:   "UpdateUser.current_password",
			data:        domain.UpdateUser{CurrentPassword: "sdf"},
			want:        "current_password must be at least 8 characters in length",
		},
		{
			description: "validate current_password greater than 30 symbols",
			fieldName:   "UpdateUser.current_password",
			data:        domain.UpdateUser{CurrentPassword: "qwertyuuioppasdfghjklzxcvbnmmasdf"},
			want:        "current_password must be a maximum of 30 characters in length",
		},
		{
			description: "validate new_password less than 8 symbols",
			fieldName:   "UpdateUser.new_password",
			data:        domain.UpdateUser{NewPassword: tests.StringPointer("sdf")},
			want:        "new_password must be at least 8 characters in length",
		},
		{
			description: "validate new_password greater than 30 symbols",
			fieldName:   "UpdateUser.new_password",
			data:        domain.UpdateUser{NewPassword: tests.StringPointer("qwertyuuioppasdfghjklzxcvbnmmasdf")},
			want:        "new_password must be a maximum of 30 characters in length",
		},
	}

	for _, tc := range casesCreateUser {
		t.Run(tc.description, func(t *testing.T) {
			if err := v.V.Struct(tc.data); err != nil {
				res := err.(validator.ValidationErrors).Translate(v.Translator)
				assert.Equal(t, tc.want, res[tc.fieldName])
			}
		})
	}

	for _, tc := range casesUpdateUser {
		t.Run(tc.description, func(t *testing.T) {
			if err := v.V.Struct(tc.data); err != nil {
				res := err.(validator.ValidationErrors).Translate(v.Translator)
				assert.Equal(t, tc.want, res[tc.fieldName])
			}
		})
	}
}
