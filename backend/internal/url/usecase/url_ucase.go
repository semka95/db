package usecase

import (
	"context"
	"time"

	"github.com/bxcodec/go-clean-arch/internal/models"
	"github.com/bxcodec/go-clean-arch/internal/url"
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
	existedURL, _ := u.GetByID(ctx, m.ID)
	if existedURL != nil {
		return models.ErrConflict
	}

	err := u.urlRepo.Store(ctx, m)
	if err != nil {
		return err
	}
	return nil
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
