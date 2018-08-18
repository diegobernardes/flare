package test

import (
	"encoding/json"
	"net/url"

	. "github.com/smartystreets/goconvey/convey" // nolint

	"github.com/diegobernardes/flare/internal"
)

func LoadResource(payload []byte) internal.Resource {
	type raw struct {
		ID       string `json:"id"`
		Endpoint string `json:"endpoint"`
		Change   struct {
			Format string `json:"format"`
			Field  string `json:"field"`
		} `json:"change"`
	}

	var r raw
	So(json.Unmarshal(payload, &r), ShouldBeNil)

	endpoint, err := url.Parse(r.Endpoint)
	So(err, ShouldBeNil)

	return internal.Resource{
		ID: r.ID,
		Change: internal.ResourceChange{
			Field:  r.Change.Field,
			Format: r.Change.Format,
		},
		Endpoint: *endpoint,
	}
}

func LoadResources(payload []byte) []internal.Resource {
	type raw []json.RawMessage

	var r raw
	So(json.Unmarshal(payload, &r), ShouldBeNil)

	resources := make([]internal.Resource, len(r))
	for i, payload := range r {
		resources[i] = LoadResource(payload)
	}
	return resources
}
