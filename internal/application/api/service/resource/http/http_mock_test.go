// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package http

import (
	"context"
	"sync"

	"github.com/diegobernardes/flare/internal"
	"github.com/diegobernardes/flare/internal/application/api/infra/http"
)

var (
	lockserviceMockCreate   sync.RWMutex
	lockserviceMockDelete   sync.RWMutex
	lockserviceMockFind     sync.RWMutex
	lockserviceMockFindByID sync.RWMutex
)

// serviceMock is a mock implementation of service.
//
//     func TestSomethingThatUsesservice(t *testing.T) {
//
//         // make and configure a mocked service
//         mockedservice := &serviceMock{
//             CreateFunc: func(ctx context.Context, resource internal.Resource) (string, error) {
// 	               panic("TODO: mock out the Create method")
//             },
//             DeleteFunc: func(ctx context.Context, resourceID string) error {
// 	               panic("TODO: mock out the Delete method")
//             },
//             FindFunc: func(ctx context.Context, pagination http.Pagination) ([]internal.Resource, http.Pagination, error) {
// 	               panic("TODO: mock out the Find method")
//             },
//             FindByIDFunc: func(ctx context.Context, resourceID string) (*internal.Resource, error) {
// 	               panic("TODO: mock out the FindByID method")
//             },
//         }
//
//         // TODO: use mockedservice in code that requires service
//         //       and then make assertions.
//
//     }
type serviceMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(ctx context.Context, resource internal.Resource) (string, error)

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(ctx context.Context, resourceID string) error

	// FindFunc mocks the Find method.
	FindFunc func(ctx context.Context, pagination http.Pagination) ([]internal.Resource, http.Pagination, error)

	// FindByIDFunc mocks the FindByID method.
	FindByIDFunc func(ctx context.Context, resourceID string) (*internal.Resource, error)

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Resource is the resource argument value.
			Resource internal.Resource
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ResourceID is the resourceID argument value.
			ResourceID string
		}
		// Find holds details about calls to the Find method.
		Find []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Pagination is the pagination argument value.
			Pagination http.Pagination
		}
		// FindByID holds details about calls to the FindByID method.
		FindByID []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ResourceID is the resourceID argument value.
			ResourceID string
		}
	}
}

// Create calls CreateFunc.
func (mock *serviceMock) Create(ctx context.Context, resource internal.Resource) (string, error) {
	if mock.CreateFunc == nil {
		panic("serviceMock.CreateFunc: method is nil but service.Create was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		Resource internal.Resource
	}{
		Ctx:      ctx,
		Resource: resource,
	}
	lockserviceMockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	lockserviceMockCreate.Unlock()
	return mock.CreateFunc(ctx, resource)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedservice.CreateCalls())
func (mock *serviceMock) CreateCalls() []struct {
	Ctx      context.Context
	Resource internal.Resource
} {
	var calls []struct {
		Ctx      context.Context
		Resource internal.Resource
	}
	lockserviceMockCreate.RLock()
	calls = mock.calls.Create
	lockserviceMockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *serviceMock) Delete(ctx context.Context, resourceID string) error {
	if mock.DeleteFunc == nil {
		panic("serviceMock.DeleteFunc: method is nil but service.Delete was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		ResourceID string
	}{
		Ctx:        ctx,
		ResourceID: resourceID,
	}
	lockserviceMockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	lockserviceMockDelete.Unlock()
	return mock.DeleteFunc(ctx, resourceID)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedservice.DeleteCalls())
func (mock *serviceMock) DeleteCalls() []struct {
	Ctx        context.Context
	ResourceID string
} {
	var calls []struct {
		Ctx        context.Context
		ResourceID string
	}
	lockserviceMockDelete.RLock()
	calls = mock.calls.Delete
	lockserviceMockDelete.RUnlock()
	return calls
}

// Find calls FindFunc.
func (mock *serviceMock) Find(ctx context.Context, pagination http.Pagination) ([]internal.Resource, http.Pagination, error) {
	if mock.FindFunc == nil {
		panic("serviceMock.FindFunc: method is nil but service.Find was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		Pagination http.Pagination
	}{
		Ctx:        ctx,
		Pagination: pagination,
	}
	lockserviceMockFind.Lock()
	mock.calls.Find = append(mock.calls.Find, callInfo)
	lockserviceMockFind.Unlock()
	return mock.FindFunc(ctx, pagination)
}

// FindCalls gets all the calls that were made to Find.
// Check the length with:
//     len(mockedservice.FindCalls())
func (mock *serviceMock) FindCalls() []struct {
	Ctx        context.Context
	Pagination http.Pagination
} {
	var calls []struct {
		Ctx        context.Context
		Pagination http.Pagination
	}
	lockserviceMockFind.RLock()
	calls = mock.calls.Find
	lockserviceMockFind.RUnlock()
	return calls
}

// FindByID calls FindByIDFunc.
func (mock *serviceMock) FindByID(ctx context.Context, resourceID string) (*internal.Resource, error) {
	if mock.FindByIDFunc == nil {
		panic("serviceMock.FindByIDFunc: method is nil but service.FindByID was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		ResourceID string
	}{
		Ctx:        ctx,
		ResourceID: resourceID,
	}
	lockserviceMockFindByID.Lock()
	mock.calls.FindByID = append(mock.calls.FindByID, callInfo)
	lockserviceMockFindByID.Unlock()
	return mock.FindByIDFunc(ctx, resourceID)
}

// FindByIDCalls gets all the calls that were made to FindByID.
// Check the length with:
//     len(mockedservice.FindByIDCalls())
func (mock *serviceMock) FindByIDCalls() []struct {
	Ctx        context.Context
	ResourceID string
} {
	var calls []struct {
		Ctx        context.Context
		ResourceID string
	}
	lockserviceMockFindByID.RLock()
	calls = mock.calls.FindByID
	lockserviceMockFindByID.RUnlock()
	return calls
}

var (
	lockserviceErrorMockAlreadyExists sync.RWMutex
	lockserviceErrorMockClient        sync.RWMutex
	lockserviceErrorMockError         sync.RWMutex
	lockserviceErrorMockNotFound      sync.RWMutex
	lockserviceErrorMockServer        sync.RWMutex
)

// serviceErrorMock is a mock implementation of serviceError.
//
//     func TestSomethingThatUsesserviceError(t *testing.T) {
//
//         // make and configure a mocked serviceError
//         mockedserviceError := &serviceErrorMock{
//             AlreadyExistsFunc: func() bool {
// 	               panic("TODO: mock out the AlreadyExists method")
//             },
//             ClientFunc: func() bool {
// 	               panic("TODO: mock out the Client method")
//             },
//             ErrorFunc: func() string {
// 	               panic("TODO: mock out the Error method")
//             },
//             NotFoundFunc: func() bool {
// 	               panic("TODO: mock out the NotFound method")
//             },
//             ServerFunc: func() bool {
// 	               panic("TODO: mock out the Server method")
//             },
//         }
//
//         // TODO: use mockedserviceError in code that requires serviceError
//         //       and then make assertions.
//
//     }
type serviceErrorMock struct {
	// AlreadyExistsFunc mocks the AlreadyExists method.
	AlreadyExistsFunc func() bool

	// ClientFunc mocks the Client method.
	ClientFunc func() bool

	// ErrorFunc mocks the Error method.
	ErrorFunc func() string

	// NotFoundFunc mocks the NotFound method.
	NotFoundFunc func() bool

	// ServerFunc mocks the Server method.
	ServerFunc func() bool

	// calls tracks calls to the methods.
	calls struct {
		// AlreadyExists holds details about calls to the AlreadyExists method.
		AlreadyExists []struct {
		}
		// Client holds details about calls to the Client method.
		Client []struct {
		}
		// Error holds details about calls to the Error method.
		Error []struct {
		}
		// NotFound holds details about calls to the NotFound method.
		NotFound []struct {
		}
		// Server holds details about calls to the Server method.
		Server []struct {
		}
	}
}

// AlreadyExists calls AlreadyExistsFunc.
func (mock *serviceErrorMock) AlreadyExists() bool {
	if mock.AlreadyExistsFunc == nil {
		panic("serviceErrorMock.AlreadyExistsFunc: method is nil but serviceError.AlreadyExists was just called")
	}
	callInfo := struct {
	}{}
	lockserviceErrorMockAlreadyExists.Lock()
	mock.calls.AlreadyExists = append(mock.calls.AlreadyExists, callInfo)
	lockserviceErrorMockAlreadyExists.Unlock()
	return mock.AlreadyExistsFunc()
}

// AlreadyExistsCalls gets all the calls that were made to AlreadyExists.
// Check the length with:
//     len(mockedserviceError.AlreadyExistsCalls())
func (mock *serviceErrorMock) AlreadyExistsCalls() []struct {
} {
	var calls []struct {
	}
	lockserviceErrorMockAlreadyExists.RLock()
	calls = mock.calls.AlreadyExists
	lockserviceErrorMockAlreadyExists.RUnlock()
	return calls
}

// Client calls ClientFunc.
func (mock *serviceErrorMock) Client() bool {
	if mock.ClientFunc == nil {
		panic("serviceErrorMock.ClientFunc: method is nil but serviceError.Client was just called")
	}
	callInfo := struct {
	}{}
	lockserviceErrorMockClient.Lock()
	mock.calls.Client = append(mock.calls.Client, callInfo)
	lockserviceErrorMockClient.Unlock()
	return mock.ClientFunc()
}

// ClientCalls gets all the calls that were made to Client.
// Check the length with:
//     len(mockedserviceError.ClientCalls())
func (mock *serviceErrorMock) ClientCalls() []struct {
} {
	var calls []struct {
	}
	lockserviceErrorMockClient.RLock()
	calls = mock.calls.Client
	lockserviceErrorMockClient.RUnlock()
	return calls
}

// Error calls ErrorFunc.
func (mock *serviceErrorMock) Error() string {
	if mock.ErrorFunc == nil {
		panic("serviceErrorMock.ErrorFunc: method is nil but serviceError.Error was just called")
	}
	callInfo := struct {
	}{}
	lockserviceErrorMockError.Lock()
	mock.calls.Error = append(mock.calls.Error, callInfo)
	lockserviceErrorMockError.Unlock()
	return mock.ErrorFunc()
}

// ErrorCalls gets all the calls that were made to Error.
// Check the length with:
//     len(mockedserviceError.ErrorCalls())
func (mock *serviceErrorMock) ErrorCalls() []struct {
} {
	var calls []struct {
	}
	lockserviceErrorMockError.RLock()
	calls = mock.calls.Error
	lockserviceErrorMockError.RUnlock()
	return calls
}

// NotFound calls NotFoundFunc.
func (mock *serviceErrorMock) NotFound() bool {
	if mock.NotFoundFunc == nil {
		panic("serviceErrorMock.NotFoundFunc: method is nil but serviceError.NotFound was just called")
	}
	callInfo := struct {
	}{}
	lockserviceErrorMockNotFound.Lock()
	mock.calls.NotFound = append(mock.calls.NotFound, callInfo)
	lockserviceErrorMockNotFound.Unlock()
	return mock.NotFoundFunc()
}

// NotFoundCalls gets all the calls that were made to NotFound.
// Check the length with:
//     len(mockedserviceError.NotFoundCalls())
func (mock *serviceErrorMock) NotFoundCalls() []struct {
} {
	var calls []struct {
	}
	lockserviceErrorMockNotFound.RLock()
	calls = mock.calls.NotFound
	lockserviceErrorMockNotFound.RUnlock()
	return calls
}

// Server calls ServerFunc.
func (mock *serviceErrorMock) Server() bool {
	if mock.ServerFunc == nil {
		panic("serviceErrorMock.ServerFunc: method is nil but serviceError.Server was just called")
	}
	callInfo := struct {
	}{}
	lockserviceErrorMockServer.Lock()
	mock.calls.Server = append(mock.calls.Server, callInfo)
	lockserviceErrorMockServer.Unlock()
	return mock.ServerFunc()
}

// ServerCalls gets all the calls that were made to Server.
// Check the length with:
//     len(mockedserviceError.ServerCalls())
func (mock *serviceErrorMock) ServerCalls() []struct {
} {
	var calls []struct {
	}
	lockserviceErrorMockServer.RLock()
	calls = mock.calls.Server
	lockserviceErrorMockServer.RUnlock()
	return calls
}
