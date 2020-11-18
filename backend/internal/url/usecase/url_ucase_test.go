package usecase_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/tests"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = sdktrace.NewTracerProvider().Tracer("")

func TestURLUsecase_GetByID(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := tests.NewURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second, tracer)

	t.Run("url not found", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, web.ErrNotFound)
		result, err := uc.GetByID(context.Background(), tURL.ID)
		assert.Error(t, err, web.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("success", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		result, err := uc.GetByID(context.Background(), tURL.ID)
		require.NoError(t, err)
		assert.EqualValues(t, tURL, result)
	})
}

func TestURLUsecase_Store(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tCreateURL := tests.NewCreateURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second, tracer)

	t.Run("success empty url ID", func(t *testing.T) {
		tCreateURL.ID = nil

		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, web.ErrNotFound)
		repository.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		require.NoError(t, err)

		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9-_]{6}$`), result.ID)
		assert.Equal(t, tCreateURL.Link, result.Link)
		assert.Equal(t, tCreateURL.ExpirationDate, result.ExpirationDate)
	})

	t.Run("success filled url ID", func(t *testing.T) {
		tCreateURL.ID = tests.StringPointer("test123456")

		repository.EXPECT().GetByID(gomock.Any(), *tCreateURL.ID).Return(nil, web.ErrNotFound)
		repository.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		require.NoError(t, err)

		assert.Equal(t, *tCreateURL.ID, result.ID)
		assert.Equal(t, tCreateURL.Link, result.Link)
		assert.Equal(t, tCreateURL.ExpirationDate, result.ExpirationDate)
	})

	t.Run("url already exists", func(t *testing.T) {
		tCreateURL.ID = tests.StringPointer("test123456")
		tURL := &models.URL{}

		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(tURL, nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		assert.Error(t, err, web.ErrConflict)
		assert.Empty(t, result)
	})

	t.Run("repository internal error", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, web.ErrNotFound)
		repository.EXPECT().Store(gomock.Any(), gomock.Any()).Return(web.ErrInternalServerError)

		result, err := uc.Store(context.Background(), tCreateURL)
		assert.Error(t, err, web.ErrInternalServerError)
		assert.Empty(t, result)
	})
}

func TestURLUsecase_Update(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tUpdateURL := tests.NewUpdateURL()
	tURL := tests.NewURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second, tracer)
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)

	t.Run("success", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		err := uc.Update(context.Background(), tUpdateURL, claims)
		require.NoError(t, err)
	})

	t.Run("url not found", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(nil, web.ErrNotFound)

		err := uc.Update(context.Background(), tUpdateURL, claims)
		assert.Error(t, err, web.ErrNotFound)
	})

	t.Run("user not authorized", func(t *testing.T) {
		claims.Subject = "wrong user"
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)

		err := uc.Update(context.Background(), tUpdateURL, claims)
		assert.Error(t, web.ErrForbidden, err)
	})

	t.Run("success by wrong user, but with admin role", func(t *testing.T) {
		claims.Roles = append(claims.Roles, auth.RoleAdmin)
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		err := uc.Update(context.Background(), tUpdateURL, claims)
		require.NoError(t, err)
	})

	t.Run("url created by not authorized user", func(t *testing.T) {
		tURL.UserID = ""
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)

		err := uc.Update(context.Background(), tUpdateURL, claims)
		assert.Error(t, web.ErrForbidden, err)
	})
}

func TestURLUsecase_Delete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := tests.NewURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second, tracer)
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)

	t.Run("success", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tURL.ID).Return(nil)
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		err := uc.Delete(context.Background(), tURL.ID, claims)
		require.NoError(t, err)
	})

	t.Run("url not found", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, web.ErrNotFound)
		err := uc.Delete(context.Background(), tURL.ID, claims)
		assert.Error(t, err, web.ErrNotFound)
	})

	t.Run("wrong user", func(t *testing.T) {
		claims.Subject = "wrong user"
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)

		err := uc.Delete(context.Background(), tURL.ID, claims)
		assert.Error(t, web.ErrForbidden, err)
	})

	t.Run("success by wrong user, but with admin role", func(t *testing.T) {
		claims.Roles = append(claims.Roles, auth.RoleAdmin)
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		repository.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

		err := uc.Delete(context.Background(), tURL.ID, claims)
		require.NoError(t, err)
	})

	t.Run("created by not authorized user", func(t *testing.T) {
		tURL.UserID = ""
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)

		err := uc.Delete(context.Background(), tURL.ID, claims)
		assert.Error(t, web.ErrForbidden, err)
	})
}
