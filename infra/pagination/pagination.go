package pagination

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

type Pagination struct {
	Limit  int
	Offset string
	Total  int
}

func (p *Pagination) Valid() error { return nil }

func (p *Pagination) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Limit  int    `json:"limit"`
		Offset string `json:"offset,omitempty"`
		Total  int    `json:"total"`
	}{p.Limit, p.Offset, p.Total})
}

// Parse extract the pagination from http.Request.
func Parse(defaultLimit int) func(r *http.Request) (*Pagination, error) {
	return func(r *http.Request) (*Pagination, error) {
		parseInt := func(key string) (int, bool, error) {
			rawValue := r.URL.Query().Get(key)
			if rawValue == "" {
				return 0, false, nil
			}

			value, err := strconv.Atoi(rawValue)
			if err != nil {
				return 0, true, errors.Wrapf(
					err, "error during parameter '%s' parse with value '%s'", key, rawValue,
				)
			}
			return value, true, nil
		}

		limit, found, err := parseInt("limit")
		if err != nil {
			return nil, err
		}
		if !found {
			limit = defaultLimit
		}

		return &Pagination{Limit: limit, Offset: r.URL.Query().Get("offset")}, nil
	}
}
