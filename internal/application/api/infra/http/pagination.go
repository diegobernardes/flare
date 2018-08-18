package http

import (
	"net/http"
	"strconv"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/internal"
)

type Pagination struct {
	Limit  uint `json:"limit"`
	Offset uint `json:"offset"`
	Total  uint `json:"total"`
}

func (p Pagination) Unmarshal() internal.Pagination {
	return internal.Pagination{
		Limit:  p.Limit,
		Offset: p.Offset,
	}
}

func (p *Pagination) Load(pagination internal.Pagination) {
	p.Limit = pagination.Limit
	p.Offset = pagination.Offset
	p.Total = pagination.Total
}

// ParsePagination extract the pagination from http.Request.
func ParsePagination(defaultLimit uint) func(r *http.Request) (*Pagination, error) {
	return func(r *http.Request) (*Pagination, error) {
		parseUint := func(key string) (uint, bool, error) {
			rawValue := r.URL.Query().Get(key)
			if rawValue == "" {
				return 0, false, nil
			}

			value, err := strconv.ParseUint(rawValue, 10, 64)
			if err != nil {
				return 0, true, errors.Wrapf(
					err, "error during parameter '%s' parse with value '%s'", key, rawValue,
				)
			}
			return uint(value), true, nil
		}

		offset, _, err := parseUint("offset")
		if err != nil {
			return nil, err
		}

		limit, found, err := parseUint("limit")
		if err != nil {
			return nil, err
		}
		if !found {
			limit = defaultLimit
		}

		return &Pagination{Limit: limit, Offset: offset}, nil
	}
}
