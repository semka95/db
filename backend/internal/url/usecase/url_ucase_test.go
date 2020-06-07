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
)

func TestURLUsecase_GetByID(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := tests.NewURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("get not existing url", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, web.ErrNotFound)
		result, err := uc.GetByID(context.Background(), tURL.ID)
		assert.Error(t, err, web.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("get url success", func(t *testing.T) {
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
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("store url empty ID", func(t *testing.T) {
		tCreateURL.ID = nil

		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, web.ErrNotFound)
		repository.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		require.NoError(t, err)

		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9-_]{6}$`), result.ID)
		assert.Equal(t, tCreateURL.Link, result.Link)
		assert.Equal(t, tCreateURL.ExpirationDate, result.ExpirationDate)
	})

	t.Run("store url filled ID", func(t *testing.T) {
		tCreateURL.ID = tests.StringPointer("test123456")

		repository.EXPECT().GetByID(gomock.Any(), *tCreateURL.ID).Return(nil, web.ErrNotFound)
		repository.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		require.NoError(t, err)

		assert.Equal(t, *tCreateURL.ID, result.ID)
		assert.Equal(t, tCreateURL.Link, result.Link)
		assert.Equal(t, tCreateURL.ExpirationDate, result.ExpirationDate)
	})

	t.Run("store existing url", func(t *testing.T) {
		tCreateURL.ID = tests.StringPointer("test123456")
		tURL := &models.URL{}

		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(tURL, nil)

		result, err := uc.Store(context.Background(), tCreateURL)
		assert.Error(t, err, web.ErrConflict)
		assert.Empty(t, result)
	})

	t.Run("store url repository error", func(t *testing.T) {
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
	uc := usecase.NewURLUsecase(repository, 10*time.Second)
	claims := auth.NewClaims("507f191e810c19729de860ea", []string{auth.RoleUser}, time.Now(), time.Minute)

	t.Run("update existing url", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		err := uc.Update(context.Background(), tUpdateURL, *claims)
		require.NoError(t, err)
	})

	t.Run("update not existing url", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(nil, web.ErrNotFound)

		err := uc.Update(context.Background(), tUpdateURL, *claims)
		assert.Error(t, err, web.ErrNotFound)
	})

	t.Run("update url wrong user", func(t *testing.T) {
		tURL.UserID = "wrong user"
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)

		err := uc.Update(context.Background(), tUpdateURL, *claims)
		assert.Error(t, web.ErrForbidden, err)
	})

	t.Run("update url wrong user, but admin", func(t *testing.T) {
		tURL.UserID = "wrong user"
		claims.Roles = append(claims.Roles, auth.RoleAdmin)
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)
		repository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		err := uc.Update(context.Background(), tUpdateURL, *claims)
		require.NoError(t, err)
	})

	t.Run("update url created by not authorized user", func(t *testing.T) {
		tURL.UserID = ""
		repository.EXPECT().GetByID(gomock.Any(), tUpdateURL.ID).Return(tURL, nil)

		err := uc.Update(context.Background(), tUpdateURL, *claims)
		assert.Error(t, web.ErrForbidden, err)
	})
}

func TestURLUsecase_Delete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := tests.NewURL()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("delete existing url", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tURL.ID).Return(nil)
		err := uc.Delete(context.Background(), tURL.ID)
		require.NoError(t, err)
	})

	t.Run("delete not existing url", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tURL.ID).Return(web.ErrNotFound)
		err := uc.Delete(context.Background(), tURL.ID)
		assert.Error(t, err, web.ErrNotFound)
	})
}
