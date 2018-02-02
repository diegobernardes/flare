// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// ParsePagination extract the pagination from http.Request.
func ParsePagination(defaultLimit int) func(r *http.Request) (*flare.Pagination, error) {
	return func(r *http.Request) (*flare.Pagination, error) {
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

		offset, _, err := parseInt("offset")
		if err != nil {
			return nil, err
		}

		limit, found, err := parseInt("limit")
		if err != nil {
			return nil, err
		}
		if !found {
			limit = defaultLimit
		}

		return &flare.Pagination{Limit: limit, Offset: offset}, nil
	}
}

// WriteResponse is used to write the response on http.ResponseWriter.
func WriteResponse(logger log.Logger) func(http.ResponseWriter, interface{}, int, http.Header) {
	logger = log.With(logger, "package", "infra/http")
	logger = level.Error(logger)

	return func(w http.ResponseWriter, r interface{}, status int, headers http.Header) {
		if headers != nil {
			for key, values := range headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}

		if r == nil {
			w.WriteHeader(status)
			return
		}

		content, err := json.Marshal(r)
		if err != nil {
			logger.Log("error", err.Error(), "message", "error during json.Marshal")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		writed, err := w.Write(content)
		if err != nil {
			logger.Log("error", err.Error(), "message", "error during write at http.ResponseWriter")
		}
		if writed != len(content) {
			logger.Log(
				"message",
				fmt.Sprintf("invalid quantity of writed bytes, expected %d and got %d", len(content), writed),
			)
		}
	}
}
