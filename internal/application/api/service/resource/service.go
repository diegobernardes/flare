package resource

import (
	"context"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
	bsResource "github.com/diegobernardes/flare/internal/service/resource"
)

type serviceRepository interface {
	Find(
		ctx context.Context, pagination internal.Pagination,
	) ([]internal.Resource, internal.Pagination, error)
	FindByID(ctx context.Context, resourceID string) (*internal.Resource, error)
	Create(ctx context.Context, resource internal.Resource) (string, error)
	Delete(ctx context.Context, resourceID string) error
}

type Service struct {
	Repository serviceRepository
}

func (s Service) Init() error {
	if s.Repository == nil {
		return errors.New("missing repository")
	}

	return nil
}

func (s Service) Find(
	ctx context.Context, rawPagination infraHTTP.Pagination,
) ([]internal.Resource, infraHTTP.Pagination, error) {
	result, response, err := s.Repository.Find(ctx, rawPagination.Unmarshal())
	if err != nil {
		return nil, rawPagination, Error{
			Cause: err, Message: "error during resource find", kind: errorKindServer,
		}
	}

	rawPagination.Load(response)
	return result, rawPagination, nil
}

func (s Service) FindByID(ctx context.Context, resourceID string) (*internal.Resource, error) {
	resource, err := s.Repository.FindByID(ctx, resourceID)
	if err != nil {
		return nil, Error{
			Cause: err, Message: "error during resource find by id", kind: errorKindServer,
		}
	}
	return resource, nil
}

func (s Service) Create(ctx context.Context, resource internal.Resource) (string, error) {
	bsResource.Init(&resource)
	bsResource.Normalize(&resource)

	if err := bsResource.Valid(resource); err != nil {
		return "", Error{Cause: err, Message: "invalid resource", kind: errorKindClient}
	}

	id, err := s.Repository.Create(ctx, resource)
	if err != nil {
		return "", Error{Cause: err, Message: "error during create", kind: errorKindServer}
	}
	return id, nil
}

func (s Service) Delete(ctx context.Context, resourceID string) error {
	if err := s.Repository.Delete(ctx, resourceID); err != nil {
		nerr, ok := err.(interface{ NotFound() bool })
		if ok && nerr.NotFound() {
			return Error{Cause: err, Message: "resource not found", kind: errorKindNotFound}
		}
		return Error{Cause: err, Message: "error during delete", kind: errorKindServer}
	}
	return nil
}
