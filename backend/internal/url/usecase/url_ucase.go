package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"bitbucket.org/dbproject_ivt/db/backend/internal/models"
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

func (u *urlUsecase) GetByID(c context.Context, id string) (*models.URL, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.urlRepo.GetByID(ctx, id)
}

func (u *urlUsecase) Update(c context.Context, updateURL *models.UpdateURL) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	url, err := u.urlRepo.GetByID(ctx, updateURL.ID)
	if err != nil {
		return fmt.Errorf("can't get %s user: %w", updateURL.ID, err)
	}

	url.ExpirationDate = updateURL.ExpirationDate
	url.UpdatedAt = time.Now().Truncate(time.Millisecond).UTC()

	return u.urlRepo.Update(ctx, url)
}

func (u *urlUsecase) Store(c context.Context, createURL *models.CreateURL) (*models.URL, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	id, err := u.getURLToken(ctx, createURL.ID)
	if err != nil {
		return nil, err
	}

	url := &models.URL{
		ID:             id,
		Link:           createURL.Link,
		ExpirationDate: createURL.ExpirationDate,
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}

	err = u.urlRepo.Store(ctx, url)
	if err != nil {
		return nil, err
	}

	return url, nil
}

func (u *urlUsecase) Delete(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.urlRepo.Delete(ctx, id)
}

func (u *urlUsecase) getURLToken(ctx context.Context, createID *string) (id string, err error) {
	if createID == nil {
		for {
			id, err = generateURLToken()
			if err != nil {
				return "", fmt.Errorf("can't create URL id token: %w", models.ErrInternalServerError)
			}
			_, err = u.GetByID(ctx, id)
			if err != nil {
				break
			}
		}
	}

	if createID != nil {
		_, err := u.GetByID(ctx, *createID)
		if err == nil {
			return "", fmt.Errorf("can't store URL, already exists: %w", models.ErrConflict)
		}
		id = *createID
	}

	return id, nil
}

func generateURLToken() (string, error) {
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(buf), nil
}
