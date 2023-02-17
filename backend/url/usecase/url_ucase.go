package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/semka95/shortener/backend/domain"
	"github.com/semka95/shortener/backend/web/auth"
)

type urlUsecase struct {
	urlRepo        domain.URLRepository
	contextTimeout time.Duration
	tracer         trace.Tracer
	urlExpiration  int
}

// NewURLUsecase will create new an urlUsecase object representation of url.Usecase interface
func NewURLUsecase(u domain.URLRepository, timeout time.Duration, tracer trace.Tracer, urlExpiration int) domain.URLUsecase {
	return &urlUsecase{
		urlRepo:        u,
		contextTimeout: timeout,
		tracer:         tracer,
		urlExpiration:  urlExpiration,
	}
}

func (uc *urlUsecase) GetByID(c context.Context, id string) (*domain.URL, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	ctx, span := uc.tracer.Start(
		ctx,
		"usecase GetByID",
		trace.WithAttributes(
			attribute.String("urlid", id)),
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u, err := uc.urlRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return u, nil
}

func (uc *urlUsecase) Update(c context.Context, updateURL domain.UpdateURL, user *auth.Claims) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	ctx, span := uc.tracer.Start(
		ctx,
		"usecase Update",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u, err := uc.urlRepo.GetByID(ctx, updateURL.ID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("can't get %s user: %w", updateURL.ID, err)
	}
	span.SetAttributes(attribute.String("urlid", updateURL.ID))

	if u.UserID == "" {
		err = fmt.Errorf("this url was created by unauthorized user: %w", domain.ErrForbidden)
		span.RecordError(err)
		return err
	}

	if !user.HasRole(auth.RoleAdmin) && u.UserID != user.Subject {
		span.RecordError(domain.ErrForbidden)
		return domain.ErrForbidden
	}

	u.ExpirationDate = updateURL.ExpirationDate
	u.UpdatedAt = time.Now().Truncate(time.Millisecond).UTC()

	err = uc.urlRepo.Update(ctx, u)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (uc *urlUsecase) Store(c context.Context, createURL domain.CreateURL) (*domain.URL, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	ctx, span := uc.tracer.Start(
		ctx,
		"usecase Store",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	id, err := uc.getURLToken(ctx, createURL.ID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("can't get %s user: %w", *createURL.ID, err)
	}

	if createURL.ExpirationDate == nil {
		expDate := time.Now().AddDate(uc.urlExpiration, 0, 0)
		createURL.ExpirationDate = &expDate
	}

	span.SetAttributes(attribute.String("urlid", id))

	u := &domain.URL{
		ID:             id,
		Link:           createURL.Link,
		ExpirationDate: *createURL.ExpirationDate,
		UserID:         createURL.UserID,
		CreatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
		UpdatedAt:      time.Now().Truncate(time.Millisecond).UTC(),
	}

	err = uc.urlRepo.Store(ctx, u)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return u, nil
}

func (uc *urlUsecase) Delete(c context.Context, id string, user *auth.Claims) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	ctx, span := uc.tracer.Start(
		ctx,
		"usecase Delete",
		trace.WithAttributes(
			attribute.String("urlid", id)),
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	u, err := uc.urlRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("can't get %s user: %w", id, err)
	}

	if u.UserID == "" {
		err = fmt.Errorf("this url was created by unauthorized user: %w", domain.ErrForbidden)
		span.RecordError(err)
		return err
	}

	if !user.HasRole(auth.RoleAdmin) && u.UserID != user.Subject {
		span.RecordError(domain.ErrForbidden)
		return domain.ErrForbidden
	}

	err = uc.urlRepo.Delete(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (uc *urlUsecase) getURLToken(ctx context.Context, createID *string) (id string, err error) {
	ctx, span := uc.tracer.Start(
		ctx,
		"usecase getURLToken",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	if createID != nil {
		_, err = uc.GetByID(ctx, *createID)
		if err == nil {
			span.RecordError(err)
			return "", fmt.Errorf("can't store URL, already exists: %w", domain.ErrConflict)
		}

		return *createID, nil
	}

	for {
		src := rand.NewSource(time.Now().UnixNano())
		id = GenerateURLToken(6, src)

		_, err = uc.GetByID(ctx, id)
		if err != nil {
			break
		}
	}

	return id, nil
}
