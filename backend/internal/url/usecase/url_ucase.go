package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/auth"
	gen "bitbucket.org/dbproject_ivt/db/backend/internal/platform/url_gen"
	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/web"
	"bitbucket.org/dbproject_ivt/db/backend/internal/url"
)

type urlUsecase struct {
	urlRepo        url.Repository
	contextTimeout time.Duration
}

// NewURLUsecase will create new an urlUsecase object representation of url.Usecase interface
func NewURLUsecase(u url.Repository, timeout time.Duration) url.Usecase {
	return &urlUsecase{
		urlRepo:        u,
		contextTimeout: timeout,
	}
}

func (uc *urlUsecase) GetByID(c context.Context, id string) (*models.URL, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	return uc.urlRepo.GetByID(ctx, id)
}

func (uc *urlUsecase) Update(c context.Context, updateURL *models.UpdateURL, user auth.Claims) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	u, err := uc.urlRepo.GetByID(ctx, updateURL.ID)
	if err != nil {
		return fmt.Errorf("can't get %s user: %w", updateURL.ID, err)
	}

	if u.UserID == "" {
		return fmt.Errorf("this url was created by unauthorized user: %w", web.ErrForbidden)
	}

	if !user.HasRole(auth.RoleAdmin) && u.UserID != user.Subject {
		return web.ErrForbidden
	}

	u.ExpirationDate = updateURL.ExpirationDate
	u.UpdatedAt = time.Now().Truncate(time.Millisecond).UTC()

	return uc.urlRepo.Update(ctx, u)
}

func (uc *urlUsecase) Store(c context.Context, createURL *models.CreateURL) (*models.URL, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	id, err := uc.getURLToken(ctx, createURL.ID)
	if err != nil {
		return nil, err
	}

	u := &models.URL{
		ID:             id,
		Link:           createURL.Link,
		ExpirationDate: createURL.ExpirationDate,
		UserID:         createURL.UserID,
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}

	err = uc.urlRepo.Store(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (uc *urlUsecase) Delete(c context.Context, id string, user auth.Claims) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	u, err := uc.urlRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("can't get %s user: %w", id, err)
	}

	if u.UserID == "" {
		return fmt.Errorf("this url was created by unauthorized user: %w", web.ErrForbidden)
	}

	if !user.HasRole(auth.RoleAdmin) && u.UserID != user.Subject {
		return web.ErrForbidden
	}

	return uc.urlRepo.Delete(ctx, id)
}

func (uc *urlUsecase) getURLToken(ctx context.Context, createID *string) (id string, err error) {
	if createID != nil {
		_, err := uc.GetByID(ctx, *createID)
		if err == nil {
			return "", fmt.Errorf("can't store URL, already exists: %w", web.ErrConflict)
		}

		return *createID, nil
	}

	for {
		src := rand.NewSource(time.Now().UnixNano())
		id = gen.GenerateURLToken(6, src)

		_, err = uc.GetByID(ctx, id)
		if err != nil {
			break
		}
	}

	return id, nil
}
