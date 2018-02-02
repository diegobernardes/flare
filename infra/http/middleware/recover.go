// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Recover is used to recover from unhandled panics.
type Recover struct {
	logger log.Logger
	writer *infraHTTP.Writer
}

// Handler process the requests and recover from any panic.
func (rec *Recover) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if errRaw := recover(); errRaw != nil {
				err := fmt.Errorf("%v", errRaw)
				level.Error(rec.logger).Log(
					"message", "unhandled error", "error", err.Error(), "stacktrace", debug.Stack(),
				)

				rec.writer.Error(w, "unhandled error", err, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// NewRecover return a configured middleware to catch panics.
func NewRecover(logger log.Logger, writer *infraHTTP.Writer) (*Recover, error) {
	if logger == nil {
		return nil, errors.New("logger not found")
	}

	if writer == nil {
		return nil, errors.New("writer not found")
	}

	return &Recover{logger, writer}, nil
}
