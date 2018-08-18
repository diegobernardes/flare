package resource

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/internal"
	"github.com/diegobernardes/flare/internal/service/shared/wildcard"
)

func Init(r *internal.Resource) {
	r.ID = uuid.NewV4().String()
}

func Normalize(r *internal.Resource) {
	r.Endpoint.Host = wildcard.Normalize(r.Endpoint.Host)
	r.Endpoint.Path = wildcard.Normalize(r.Endpoint.Path)

	if r.Endpoint.Path == "" {
		return
	}

	if r.Endpoint.Path[len(r.Endpoint.Path)-1] == '/' {
		r.Endpoint.Path = r.Endpoint.Path[:len(r.Endpoint.Path)-1]
	}
}

func Valid(r internal.Resource) error {
	if err := validEndpoint(r.Endpoint); err != nil {
		return errors.Wrap(err, "invalid endpoint")
	}

	if err := validEndpointWildcard(r.Endpoint); err != nil {
		return errors.Wrap(err, "invalid endpoint")
	}

	if r.Change.Field == "" {
		return errors.New("missing change field")
	}

	return nil
}

func validEndpoint(endpoint url.URL) error {
	if endpoint.Opaque != "" {
		return fmt.Errorf("should not have opaque content '%s'", endpoint.Opaque)
	}

	if endpoint.User != nil {
		return errors.New("should not have user")
	}

	if endpoint.Host == "" {
		return errors.New("missing host")
	}

	if endpoint.Path == "" {
		return errors.New("missing path")
	}

	if endpoint.RawQuery != "" {
		return fmt.Errorf("should not have query string '%s'", endpoint.RawQuery)
	}

	if endpoint.Fragment != "" {
		return fmt.Errorf("should not have fragment '%s'", endpoint.Fragment)
	}

	switch endpoint.Scheme {
	case "http", "https":
	case "":
		return errors.New("missing scheme")
	default:
		return errors.New("unknown scheme")
	}

	return nil
}

func validEndpointWildcard(endpoint url.URL) error {
	if !wildcard.Present(endpoint.Path) {
		return errors.New("missing wildcard")
	}

	if err := wildcard.Valid(endpoint.Path); err != nil {
		return errors.Wrap(err, "can't have duplicated wildcards")
	}

	return nil
}
