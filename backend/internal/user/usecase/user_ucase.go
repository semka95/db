package usecase

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
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

func (uc *userUsecase) GetByID(c context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("user ID is not valid ObjectID: %w: %s", web.ErrBadParamInput, err.Error())
	}

	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	return uc.userRepo.GetByID(ctx, objID)
}

func (uc *userUsecase) Update(c context.Context, updateUser *models.UpdateUser, claims auth.Claims) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	u, err := uc.userRepo.GetByID(ctx, updateUser.ID)
	if err != nil {
		return fmt.Errorf("can't get %s user: %w", updateUser.ID.Hex(), err)
	}

	if !claims.HasRole(auth.RoleAdmin) && u.ID.Hex() != claims.Subject {
		return web.ErrForbidden
	}

	if updateUser.FullName != nil {
		u.FullName = *updateUser.FullName
	}

	if updateUser.Email != nil {
		u.Email = *updateUser.Email
	}

	if updateUser.Password != nil {
		hashedPwd, err := generateHash(*updateUser.Password)
		if err != nil {
			return fmt.Errorf("can't generate hash from this password - %s: %w: %s", *updateUser.Password, web.ErrInternalServerError, err.Error())
		}
		u.HashedPassword = hashedPwd
	}

	u.UpdatedAt = time.Now().Truncate(time.Millisecond).UTC()

	return uc.userRepo.Update(ctx, u)
}

func (uc *userUsecase) Create(c context.Context, m *models.CreateUser) (*models.User, error) {
	hashedPwd, err := generateHash(m.Password)
	if err != nil {
		return nil, fmt.Errorf("can't generate hash from this password - %s: %w: %s", m.Password, web.ErrInternalServerError, err.Error())
	}

	u := &models.User{
		ID:             primitive.NewObjectID(),
		FullName:       m.FullName,
		Email:          m.Email,
		HashedPassword: hashedPwd,
		Roles:          []string{auth.RoleUser},
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}

	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	err = uc.userRepo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (uc *userUsecase) Delete(c context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("user ID is not valid ObjectID: %w: %s", web.ErrBadParamInput, err.Error())
	}

	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	return uc.userRepo.Delete(ctx, objID)
}

func (uc *userUsecase) Authenticate(c context.Context, now time.Time, email, password string) (*auth.Claims, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	u, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", web.ErrAuthenticationFailure, err.Error())
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("compare password error: %w: %s", web.ErrAuthenticationFailure, err.Error())
	}

	claims := auth.NewClaims(u.ID.Hex(), u.Roles, now, time.Hour)
	return claims, nil
}

func generateHash(pass string) (string, error) {
	result, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
