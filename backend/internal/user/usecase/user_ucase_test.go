package usecase_test

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user/mocks"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user/usecase"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestUserUsecase_GetByID(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tURL := models.NewUser()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewUserUsecase(repository, 10*time.Second)

	t.Run("get not valid id", func(t *testing.T) {
		result, err := uc.GetByID(context.Background(), "not valid id")
		assert.Error(t, err, models.ErrBadParamInput)
		assert.Nil(t, result)
	})

	t.Run("get not existed user", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(nil, models.ErrNotFound)
		result, err := uc.GetByID(context.Background(), tURL.ID.Hex())
		assert.Error(t, err, models.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("get user success", func(t *testing.T) {
		repository.EXPECT().GetByID(gomock.Any(), tURL.ID).Return(tURL, nil)
		result, err := uc.GetByID(context.Background(), tURL.ID.Hex())
		assert.NoError(t, err)
		assert.EqualValues(t, tURL, result)
	})
}

func TestUserUsecase_Update(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tUser := models.NewUser()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewUserUsecase(repository, 10*time.Second)

	t.Run("update user success", func(t *testing.T) {
		repository.EXPECT().Update(gomock.Any(), tUser).Return(nil)
		err := uc.Update(context.Background(), tUser)
		assert.NoError(t, err)
	})

	t.Run("update not existed user", func(t *testing.T) {
		repository.EXPECT().Update(gomock.Any(), tUser).Return(models.ErrNotFound)
		err := uc.Update(context.Background(), tUser)
		assert.Error(t, err, models.ErrNotFound)
	})
}

func TestUserUsecase_Create(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tUser := models.NewUser()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewUserUsecase(repository, 10*time.Second)

	t.Run("create user error", func(t *testing.T) {
		repository.EXPECT().Create(gomock.Any(), tUser).Return(models.ErrInternalServerError)
		result, err := uc.Create(context.Background(), tUser)
		assert.Error(t, err, models.ErrInternalServerError)
		assert.Empty(t, result)
	})

	t.Run("create user success", func(t *testing.T) {
		repository.EXPECT().Create(gomock.Any(), tUser).Return(nil)
		result, err := uc.Create(context.Background(), tUser)
		assert.NoError(t, err)
		assert.Equal(t, tUser.ID.Hex(), result)
	})
}

func TestUserUsecase_Delete(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tUser := models.NewUser()

	repository := mocks.NewMockRepository(controller)
	uc := usecase.NewUserUsecase(repository, 10*time.Second)

	t.Run("delete not valid id", func(t *testing.T) {
		err := uc.Delete(context.Background(), "not valid id")
		assert.Error(t, err, models.ErrBadParamInput)
	})

	t.Run("delete not existed user", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tUser.ID).Return(models.ErrNoAffected)
		err := uc.Delete(context.Background(), tUser.ID.Hex())
		assert.Error(t, err, models.ErrNoAffected)
	})

	t.Run("delete success", func(t *testing.T) {
		repository.EXPECT().Delete(gomock.Any(), tUser.ID).Return(nil)
		err := uc.Delete(context.Background(), tUser.ID.Hex())
		assert.NoError(t, err)
	})
}
