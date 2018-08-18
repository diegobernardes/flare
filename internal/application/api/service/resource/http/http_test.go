package http

import (
	"encoding/json"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
	"github.com/diegobernardes/flare/internal/infra/test"
)

func genError(kind string) error {
	return &serviceErrorMock{
		ServerFunc:        func() bool { return kind == "server" },
		AlreadyExistsFunc: func() bool { return kind == "alreadyExists" },
		ClientFunc:        func() bool { return kind == "client" },
		NotFoundFunc:      func() bool { return kind == "notFound" },
		ErrorFunc:         func() string { return "custom error" },
	}
}

func loadIndexResponse(payload []byte) ([]internal.Resource, infraHTTP.Pagination) {
	type raw struct {
		Pagination struct {
			Limit  uint `json:"limit"`
			Offset uint `json:"offset"`
			Total  uint `json:"total"`
		} `json:"pagination"`
		Resources []json.RawMessage `json:"resources"`
	}

	var r raw
	So(json.Unmarshal(payload, &r), ShouldBeNil)

	resources := make([]internal.Resource, len(r.Resources))
	for i, resourcePayload := range r.Resources {
		resources[i] = test.LoadResource(resourcePayload)
	}

	return resources, infraHTTP.Pagination{
		Limit:  r.Pagination.Limit,
		Offset: r.Pagination.Offset,
		Total:  r.Pagination.Total,
	}
}
