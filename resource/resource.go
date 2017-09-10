package resource

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
)

type pagination struct {
	base *flare.Pagination
}

func (p *pagination) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}{
		Limit:  p.base.Limit,
		Total:  p.base.Total,
		Offset: p.base.Offset,
	})
}

type resource struct {
	base *flare.Resource
}

func (r *resource) MarshalJSON() ([]byte, error) {
	change := map[string]string{
		"kind":  r.base.Change.Kind,
		"field": r.base.Change.Field,
	}

	if r.base.Change.DateFormat != "" {
		change["dateFormat"] = r.base.Change.DateFormat
	}

	return json.Marshal(&struct {
		Id        string            `json:"id"`
		Domains   []string          `json:"domains"`
		Path      string            `json:"path"`
		Change    map[string]string `json:"change"`
		CreatedAt string            `json:"createdAt"`
	}{
		Id:        r.base.Id,
		Domains:   r.base.Domains,
		Path:      r.base.Path,
		Change:    change,
		CreatedAt: r.base.CreatedAt.Format(time.RFC3339),
	})
}

type resourceCreate struct {
	Path    string   `json:"path"`
	Domains []string `json:"domains"`
	Change  struct {
		Kind       string `json:"kind"`
		Field      string `json:"field"`
		DateFormat string `json:"dateFormat"`
	} `json:"change"`
}

func (r *resourceCreate) cleanup() {
	trim := func(value string) string { return strings.TrimSpace(value) }
	r.Path = trim(r.Path)
	r.Change.Kind = trim(r.Change.Kind)
	r.Change.Field = trim(r.Change.Field)
	r.Change.DateFormat = trim(r.Change.DateFormat)

	for i, value := range r.Domains {
		r.Domains[i] = trim(value)
	}
}

func (r *resourceCreate) valid() error {
	if len(r.Domains) == 0 {
		return errors.New("missing domains")
	}

	if r.Path == "" {
		return errors.New("missing path")
	}

	if err := r.validTrack(); err != nil {
		return err
	}

	if err := r.validWildcard(); err != nil {
		return err
	}

	if r.Change.Field == "" {
		return errors.New("missing change")
	}

	switch r.Change.Kind {
	case flare.ResourceChangeInteger, flare.ResourceChangeString:
	case flare.ResourceChangeDate:
		if r.Change.DateFormat == "" {
			return errors.New("missing change.dateFormat")
		}
	default:
		return errors.New("invalid change.kind")
	}

	return nil
}

func (r *resourceCreate) validTrack() error {
	trackCount := strings.Count(r.Path, "{track}")
	if trackCount == 0 {
		return errors.New("missing track mark on path")
	} else if trackCount > 1 {
		return errors.Errorf("path can have only one track, current has %d", trackCount)
	}

	trackSuffix := r.Path[strings.Index(r.Path, "{track}"):]
	if strings.Contains(trackSuffix, "{*}") {
		return errors.New("found wildcard after the track mark")
	}
	return nil
}

func (r *resourceCreate) validWildcard() error {
	for _, value := range strings.Split(r.Path, "/") {
		if strings.Contains(value, "{*}") && value != "{*}" {
			return errors.New("found a mixed wildcard on path, it should appear alone")
		}
	}
	return nil
}

func (r *resourceCreate) toFlareResource() *flare.Resource {
	return &flare.Resource{
		Id:      uuid.NewV4().String(),
		Domains: r.Domains,
		Path:    r.Path,
		Change: flare.ResourceChange{
			Kind:       r.Change.Kind,
			Field:      r.Change.Field,
			DateFormat: r.Change.DateFormat,
		},
	}
}

func transformResource(r *flare.Resource) *resource { return &resource{r} }

func transformPagination(p *flare.Pagination) *pagination { return &pagination{base: p} }

func transformResources(r []flare.Resource) []resource {
	result := make([]resource, len(r))
	for i := 0; i < len(r); i++ {
		result[i] = resource{&r[i]}
	}
	return result
}

type response struct {
	Pagination *pagination
	Error      *responseError
	Resources  []resource
	Resource   *resource
}

func (r *response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Error != nil {
		result = map[string]*responseError{"error": r.Error}
	} else if r.Resource != nil {
		result = r.Resource
	} else {
		result = map[string]interface{}{
			"pagination": r.Pagination,
			"resources":  r.Resources,
		}
	}

	return json.Marshal(result)
}

type responseError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}
