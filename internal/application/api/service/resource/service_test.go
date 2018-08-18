//go:generate moq -out resource_mock_test.go . serviceRepository
package resource

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare/internal"
	"github.com/diegobernardes/flare/internal/application/api/infra/http"
	infraTest "github.com/diegobernardes/flare/internal/infra/test"
)

func TestServiceInit(t *testing.T) {
	Convey("Feature: Service initialization", t, func() {
		Convey("Given a list services", func() {
			tests := []struct {
				title       string
				service     Service
				shouldError bool
			}{
				{
					"have a missing repository error",
					Service{},
					true,
				},
				{
					"success",
					Service{Repository: &serviceRepositoryMock{}},
					false,
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					err := tt.service.Init()
					if tt.shouldError {
						So(err, ShouldNotBeNil)
					} else {
						So(err, ShouldBeNil)
					}
				})
			}
		})
	})
}

func TestServiceFind(t *testing.T) {
	Convey("Feature: Call the service list find resources", t, func() {
		Convey("Given a list of services", func() {
			tests := []struct {
				title      string
				repository serviceRepositoryMock
				resources  []internal.Resource
				pagination http.Pagination
				shouldErr  bool
				errKind    string
			}{
				{
					"have a error",
					serviceRepositoryMock{
						FindFunc: func(
							ctx context.Context, pagination internal.Pagination,
						) ([]internal.Resource, internal.Pagination, error) {
							return nil, internal.Pagination{}, errors.New("custom error")
						},
					},
					nil,
					http.Pagination{},
					true,
					errorKindServer,
				},
				{
					"success",
					serviceRepositoryMock{
						FindFunc: func(
							ctx context.Context, pagination internal.Pagination,
						) ([]internal.Resource, internal.Pagination, error) {
							resources := infraTest.LoadResources(infraTest.Load("resources.json"))
							return resources, internal.Pagination{Total: (uint)(len(resources))}, nil
						},
					},
					infraTest.LoadResources(infraTest.Load("resources.json")),
					http.Pagination{Total: 2},
					false,
					"",
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					service := Service{Repository: &tt.repository}
					So(service.Init(), ShouldBeNil)

					resources, pagination, err := service.Find(context.Background(), http.Pagination{})
					So(err != nil, ShouldEqual, tt.shouldErr)
					if err != nil {
						So(err.(Error).kind, ShouldEqual, tt.errKind)
					}
					So(pagination, ShouldResemble, tt.pagination)
					So(resources, ShouldResemble, tt.resources)
				})
			}
		})
	})
}

func TestServiceFindByID(t *testing.T) {
	Convey("Feature: Call the service find a resource by id", t, func() {
		Convey("Given a list of services", func() {
			tests := []struct {
				title        string
				repository   serviceRepositoryMock
				testdataPath string
				shouldErr    bool
				errKind      string
			}{
				{
					"have a error",
					serviceRepositoryMock{
						FindByIDFunc: func(context.Context, string) (*internal.Resource, error) {
							return nil, errors.New("custom error")
						},
					},
					"",
					true,
					errorKindServer,
				},
				{
					"success",
					serviceRepositoryMock{
						FindByIDFunc: func(context.Context, string) (*internal.Resource, error) {
							resource := infraTest.LoadResource(infraTest.Load("resource.json"))
							return &resource, nil
						},
					},
					"resource.json",
					false,
					"",
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					service := Service{Repository: &tt.repository}
					So(service.Init(), ShouldBeNil)

					resource, err := service.FindByID(context.Background(), "123")
					So(err != nil, ShouldResemble, tt.shouldErr)
					if err != nil {
						So(err.(Error).kind, ShouldEqual, tt.errKind)
					}

					if tt.testdataPath != "" {
						testdataResource := infraTest.LoadResource(infraTest.Load(tt.testdataPath))
						So(resource, ShouldResemble, &testdataResource)
					}
				})
			}
		})
	})
}

func TestServiceCreate(t *testing.T) {
	Convey("Feature: Call the service to create a resource", t, func() {
		Convey("Given a list of services", func() {
			tests := []struct {
				title        string
				repository   serviceRepositoryMock
				testdataPath string
				shouldErr    bool
				errKind      string
			}{
				{
					"have a client error",
					serviceRepositoryMock{
						CreateFunc: func(context.Context, internal.Resource) (string, error) {
							return "", errors.New("custom error")
						},
					},
					"resource.invalid.json",
					true,
					errorKindClient,
				},
				{
					"have a server error",
					serviceRepositoryMock{
						CreateFunc: func(context.Context, internal.Resource) (string, error) {
							return "", errors.New("custom error")
						},
					},
					"resource.json",
					true,
					errorKindServer,
				},
				{
					"success",
					serviceRepositoryMock{
						CreateFunc: func(_ context.Context, r internal.Resource) (string, error) {
							return r.ID, nil
						},
					},
					"resource.json",
					false,
					"",
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					service := Service{Repository: &tt.repository}
					So(service.Init(), ShouldBeNil)

					resource := infraTest.LoadResource(infraTest.Load(tt.testdataPath))
					id, err := service.Create(context.Background(), resource)
					So(err != nil, ShouldResemble, tt.shouldErr)
					if err == nil {
						So(id, ShouldNotEqual, "")
					} else {
						So(err.(Error).kind, ShouldEqual, tt.errKind)
					}
				})
			}
		})
	})
}

func TestServiceDelete(t *testing.T) {
	Convey("Feature: Call the service to delete a resource", t, func() {
		Convey("Given a list of services", func() {
			tests := []struct {
				title      string
				repository serviceRepositoryMock
				shouldErr  bool
				errKind    string
			}{
				{
					"return a not found error",
					serviceRepositoryMock{
						DeleteFunc: func(context.Context, string) error {
							return Error{kind: errorKindNotFound}
						},
					},
					true,
					errorKindNotFound,
				},
				{
					"return a server error",
					serviceRepositoryMock{
						DeleteFunc: func(context.Context, string) error { return Error{kind: errorKindServer} },
					},
					true,
					errorKindServer,
				},
				{
					"success",
					serviceRepositoryMock{DeleteFunc: func(context.Context, string) error { return nil }},
					false,
					"",
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					service := Service{Repository: &tt.repository}
					So(service.Init(), ShouldBeNil)

					err := service.Delete(context.Background(), "123")
					So(err != nil, ShouldResemble, tt.shouldErr)
					if err != nil {
						So(err.(Error).kind, ShouldEqual, tt.errKind)
					}
				})
			}
		})
	})
}
