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

func (u *urlUsecase) Update(c context.Context, url *models.URL) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.urlRepo.Update(ctx, url)
}

func (u *urlUsecase) Store(c context.Context, m *models.URL) (string, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()
	if m.ID == "" {
		id, err := createURLToken()
		if err != nil {
			return "", fmt.Errorf("Can't create URL id token: %w", models.ErrInternalServerError)
		}
		// refactor
		for {
			_, err := u.GetByID(ctx, id)
			if err != nil {
				break
			}
			id, err = createURLToken()
			if err != nil {
				return "", fmt.Errorf("Can't create URL id token: %w", models.ErrInternalServerError)
			}
		}
		m.ID = id
	} else {
		_, err := u.GetByID(ctx, m.ID)
		if err == nil {
			return "", fmt.Errorf("Can't store URL, already exists: %w", models.ErrConflict)
		}
	}

	err := u.urlRepo.Store(ctx, m)
	if err != nil {
		return "", err
	}

	return m.ID, nil
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

	return u.urlRepo.Delete(ctx, id)
}
