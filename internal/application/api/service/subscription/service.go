package subscription

import (
	"context"
	"errors"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
)

type serviceRepository interface {
	Find(
		ctx context.Context, pagination internal.Pagination,
	) ([]internal.Subscription, internal.Pagination, error)
	FindByID(ctx context.Context, subscriptionID string) (*internal.Subscription, error)
	Create(ctx context.Context, subscription internal.Subscription) (string, error)
	Delete(ctx context.Context, subscriptionID string) error
}

type Service struct {
	Repository serviceRepository
}

func (s Service) Init() error {
	if s.Repository == nil {
		return errors.New("missing iterator")
	}
	return nil
}

func (s Service) Find(
	ctx context.Context, pagination infraHTTP.Pagination,
) ([]internal.Subscription, infraHTTP.Pagination, error) {
	subscriptions, rawPagination, err := s.Repository.Find(ctx, pagination.Unmarshal())
	pagination.Load(rawPagination)
	return subscriptions, pagination, err
}

func (s Service) FindByID(ctx context.Context, subscriptionID string) (*internal.Subscription, error) {
	return s.Repository.FindByID(ctx, subscriptionID)
}

func (s Service) Create(ctx context.Context, subscription internal.Subscription) (string, error) {
	return s.Repository.Create(ctx, subscription)
}

func (s Service) Delete(ctx context.Context, subscriptionID string) error {
	return s.Repository.Delete(ctx, subscriptionID)
}

func (s Service) FindResource(ctx context.Context, resourceID string) (*internal.Resource, error) {
	return nil, nil
}
