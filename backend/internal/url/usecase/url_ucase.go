package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

	res, err := u.urlRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *urlUsecase) Update(c context.Context, url *models.URL) error {

	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.urlRepo.Update(ctx, url)
}

func (u *urlUsecase) Store(c context.Context, m *models.URL) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()
	if m.ID == "" {
		id, err := createURLToken()
		if err != nil {
			return err
		}
		// refactor
		for {
			existedURL, _ := u.GetByID(ctx, id)
			if existedURL == nil {
				break
			}
			id, err = createURLToken()
			if err != nil {
				return err
			}
		}
		m.ID = id
	} else {
		existedURL, _ := u.GetByID(ctx, m.ID)
		if existedURL != nil {
			return models.ErrConflict
		}
	}

	err := u.urlRepo.Store(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func createURLToken() (string, error) {
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(buf), nil
}

func (u *urlUsecase) Delete(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()
	existedURL, err := u.urlRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existedURL == nil {
		return models.ErrNotFound
	}
	return u.urlRepo.Delete(ctx, id)
}
