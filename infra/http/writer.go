// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

// Writer is used to send the content to the client.
type Writer struct {
	logger log.Logger
}

// Response is used write response on http.ResponseWriter.
func (wrt *Writer) Response(w http.ResponseWriter, r interface{}, status int, headers http.Header) {
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
		wrt.logger.Log("error", err.Error(), "message", "error during json.Marshal")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	writed, err := w.Write(content)
	if err != nil {
		wrt.logger.Log("error", err.Error(), "message", "error during write at http.ResponseWriter")
	}
	if writed != len(content) {
		wrt.logger.Log(
			"message",
			fmt.Sprintf("invalid quantity of writed bytes, expected %d and got %d", len(content), writed),
		)
	}
}

// Error is used to generate a proper error content to be sent to the client.
func (wrt *Writer) Error(w http.ResponseWriter, title string, err error, status int) {
	resp := struct {
		Error struct {
			Title  string `json:"title"`
			Detail string `json:"detail,omitempty"`
		} `json:"error"`
	}{}

	if err != nil {
		resp.Error.Detail = err.Error()
	}

	if title != "" {
		resp.Error.Title = title
	}

	wrt.Response(w, &resp, status, nil)
}

// NewWriter returns a configured writer.
func NewWriter(logger log.Logger) (*Writer, error) {
	if logger == nil {
		return nil, errors.New("logger not found")
	}
	logger = log.With(logger, "package", "infra/http")
	logger = level.Error(logger)
	return &Writer{logger}, nil
}
