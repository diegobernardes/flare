// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Log is a middleware to log the requests.
type Log struct {
	logger log.Logger
}

// Handler process and log requests.
func (l *Log) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		reqId := middleware.GetReqID(r.Context())
		l.logger.Log(
			"requestId", reqId,
			"method", r.Method,
			"endpoint", r.RequestURI,
			"protocol", r.Proto,
			"message", "request started",
		)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		l.logger.Log([]interface{}{
			"requestId", reqId,
			"duration", time.Since(t1),
			"contentLength", ww.BytesWritten(),
			"status", ww.Status(),
			"message", "request finished",
		}...)
	}

	return http.HandlerFunc(fn)
}

// NewLog return a configured middleware to log the requests.
func NewLog(logger log.Logger) Log {
	logger = log.With(logger, "package", "infra/http/middleware")
	logger = level.Info(logger)
	return Log{logger}
}
