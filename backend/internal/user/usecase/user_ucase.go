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
		return nil, fmt.Errorf("User ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
	}

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	user, err := u.userRepo.GetByID(ctx, objID)
	if err == nil {
		user.Sanitize()
	}

	return user, err
}

func (u *userUsecase) Update(c context.Context, user *models.User) error {
	hashedPwd, err := generateHash(user.Password)
	if err != nil {
		return fmt.Errorf("Can't generate hash from this password - %s: %w: %s", user.Password, models.ErrInternalServerError, err.Error())
	}
	user.Password = hashedPwd

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.userRepo.Update(ctx, user)
}

func (u *userUsecase) Create(c context.Context, m *models.User) (string, error) {
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now().Truncate(time.Millisecond).UTC()
	hashedPwd, err := generateHash(m.Password)
	if err != nil {
		return "", fmt.Errorf("Can't generate hash from this password - %s: %w: %s", m.Password, models.ErrInternalServerError, err.Error())
	}
	m.Password = hashedPwd

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	err = u.userRepo.Create(ctx, m)
	if err != nil {
		return "", err
	}

	return m.ID.Hex(), nil
}

func (u *userUsecase) Delete(c context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("User ID is not valid ObjectID: %w: %s", models.ErrBadParamInput, err.Error())
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
