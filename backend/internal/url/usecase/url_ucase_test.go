package usecase_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/mocks"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestURLUsecase_GetByID(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := &models.URL{
		ID:             "test",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("test get not existed record", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, models.ErrNotFound)
		result, err := uc.GetByID(context.Background(), tURL.ID)
		assert.Error(t, err, models.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("test get existed record", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		result, err := uc.GetByID(context.Background(), tURL.ID)
		assert.NoError(t, err)
		assert.ObjectsAreEqual(tURL, result)
	})
}

func TestURLUsecase_Store(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := &models.URL{
		ID:             "test",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("test store empty ID", func(t *testing.T) {
		tURL.ID = ""
		repository.EXPECT().Store(gomock.Any(), tURL).Return(nil)
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, models.ErrNotFound)
		result, err := uc.Store(context.Background(), tURL)
		assert.NoError(t, err)
		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]{6}$`), result)
	})

	t.Run("test store empty ID, generated existed token", func(t *testing.T) {
		tURL.ID = ""
		repository.EXPECT().Store(gomock.Any(), tURL).Return(nil)
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(tURL, nil).Times(1)
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, models.ErrNotFound)
		result, err := uc.Store(context.Background(), tURL)
		assert.NoError(t, err)
		assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]{6}$`), result)
	})

	t.Run("test store filled ID", func(t *testing.T) {
		tURL.ID = "test"
		repository.EXPECT().Store(gomock.Any(), tURL).Return(nil)
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, models.ErrNotFound)
		result, err := uc.Store(context.Background(), tURL)
		assert.NoError(t, err)
		assert.Equal(t, tURL.ID, result)
	})

	t.Run("test store already existed ID", func(t *testing.T) {
		tURL.ID = "test"
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(tURL, nil)
		result, err := uc.Store(context.Background(), tURL)
		assert.Error(t, err, models.ErrConflict)
		assert.Empty(t, result)
	})

	t.Run("test store repository store error", func(t *testing.T) {
		tURL.ID = "test"
		repository.EXPECT().Store(gomock.Any(), tURL).Return(models.ErrInternalServerError)
		repository.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, models.ErrNotFound)
		result, err := uc.Store(context.Background(), tURL)
		assert.Error(t, err, models.ErrInternalServerError)
		assert.Empty(t, result)
	})
}

func TestURLUsecase_Update(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := &models.URL{
		ID:             "test",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("test update existed record", func(t *testing.T) {
		repository.EXPECT().Update(gomock.Any(), tURL).Return(nil)
		err := uc.Update(context.Background(), tURL)
		assert.NoError(t, err)
	})

	t.Run("test update not existed record", func(t *testing.T) {
		repository.EXPECT().Update(gomock.Any(), tURL).Return(models.ErrNotFound)
		err := uc.Update(context.Background(), tURL)
		assert.Error(t, err, models.ErrNotFound)
	})
}

func TestURLUsecase_Delete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := &models.URL{
		ID:             "test",
		Link:           "http://www.example.org",
		ExpirationDate: time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewURLUsecase(repository, 10*time.Second)

	t.Run("test delete existed record", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tURL.ID).Return(nil)
		err := uc.Delete(context.Background(), tURL.ID)
		assert.NoError(t, err)
	})

	t.Run("test delete not existed record", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tURL.ID).Return(models.ErrNotFound)
		err := uc.Delete(context.Background(), tURL.ID)
		assert.Error(t, err, models.ErrNotFound)
	})
}
