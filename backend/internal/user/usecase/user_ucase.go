package usecase

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type userUsecase struct {
	userRepo       user.Repository
	contextTimeout time.Duration
}

// NewUserUsecase will create new an userUsecase object representation of user.Usecase interface
func NewUserUsecase(u user.Repository, timeout time.Duration) user.Usecase {
	return &userUsecase{
		userRepo:       u,
		contextTimeout: timeout,
	}
}

func (u *userUsecase) GetByID(c context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("user ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
	}

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.userRepo.GetByID(ctx, objID)
}

func (u *userUsecase) Update(c context.Context, updateUser *models.UpdateUser) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	user, err := u.userRepo.GetByID(ctx, updateUser.ID)
	if err != nil {
		return fmt.Errorf("can't get %s user: %w", updateUser.ID.Hex(), err)
	}

	if updateUser.FullName != nil {
		user.FullName = *updateUser.FullName
	}

	if updateUser.Email != nil {
		user.Email = *updateUser.Email
	}

	if updateUser.Password != nil {
		hashedPwd, err := generateHash(*updateUser.Password)
		if err != nil {
			return fmt.Errorf("can't generate hash from this password - %s: %w: %s", *updateUser.Password, models.ErrInternalServerError, err.Error())
		}
		user.HashedPassword = hashedPwd
	}

	user.UpdatedAt = time.Now().Truncate(time.Millisecond).UTC()

	return u.userRepo.Update(ctx, user)
}

func (u *userUsecase) Create(c context.Context, m *models.CreateUser) (*models.User, error) {
	hashedPwd, err := generateHash(m.Password)
	if err != nil {
		return nil, fmt.Errorf("can't generate hash from this password - %s: %w: %s", m.Password, models.ErrInternalServerError, err.Error())
	}

	user := &models.User{
		ID:             primitive.NewObjectID(),
		FullName:       m.FullName,
		Email:          m.Email,
		HashedPassword: hashedPwd,
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	err = u.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) Delete(c context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("user ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
	}

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.userRepo.Delete(ctx, objID)
}

func generateHash(pass string) (string, error) {
	result, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
