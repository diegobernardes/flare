// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	document "github.com/diegobernardes/flare/domain/document/http"
	resource "github.com/diegobernardes/flare/domain/resource/http"
	subscription "github.com/diegobernardes/flare/domain/subscription/http"
	"github.com/diegobernardes/flare/infra/config"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	infraMiddleware "github.com/diegobernardes/flare/infra/http/middleware"
)

type server struct {
	cfg        *config.Client
	addr       string
	httpServer http.Server
	handler    struct {
		resource     *resource.Handler
		subscription *subscription.Handler
		document     *document.Handler
	}
	middleware struct {
		timeout time.Duration
	}
	logger        log.Logger
	errLogger     log.Logger
	writeResponse func(http.ResponseWriter, interface{}, int, http.Header)
}

func (s *server) start() error {
	router, err := s.router()
	if err != nil {
		return errors.Wrap(err, "error during router initialization")
	}

	s.httpServer = http.Server{
		Addr:              s.addr,
		Handler:           router,
		ReadHeaderTimeout: s.middleware.timeout * 2,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			s.errLogger.Log("error", err.Error(), "message", "error during server initialization")

			// As the listen is running on a goroutine, if it returns any error like the port already
			// being used, the application don't gonna exit. To trigger the exit, we are sending a
			// interrupt signal to the process.
			process, err := os.FindProcess(os.Getpid())
			if err != nil {
				s.errLogger.Log("error", err.Error(), "message", "couldn't find Flare process to exit")
				os.Exit(1)
			}
			if err := process.Signal(os.Interrupt); err != nil {
				s.errLogger.Log("error", err.Error(), "message", "error during signal Flare process to exit")
				os.Exit(1)
			}
		}
	}()

	return nil
}

func (s *server) stop() error {
	level.Info(s.logger).Log("message", "waiting the remaining connections to complete")
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		return errors.Wrap(err, "error during server close")
	}
	return nil
}

func (s *server) router() (http.Handler, error) {
	r := chi.NewRouter()
	if err := s.initMiddleware(r); err != nil {
		return nil, errors.Wrap(err, "error during middleware initialization")
	}

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		s.writeResponse(w, map[string]interface{}{
			"error": map[string]interface{}{
				"status": http.StatusBadRequest,
				"title":  "method not allowed",
			},
		}, http.StatusBadRequest, nil)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.writeResponse(w, map[string]interface{}{
			"error": map[string]interface{}{
				"status": http.StatusNotFound,
				"title":  "not found",
			},
		}, http.StatusNotFound, nil)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		content := map[string]string{"go": GoVersion}

		if Version != "" {
			content["version"] = Version
		}

		if Commit != "" {
			content["commit"] = Commit
		}

		if BuildTime != "" {
			content["buildTime"] = BuildTime
		}

		s.writeResponse(w, content, http.StatusOK, nil)
	})

	r.Route("/resources", s.routerResource)
	r.Route("/resources/{resourceID}/subscriptions", s.routerSubscription)
	r.Route("/documents", s.routerDocument)

	return r, nil
}

func (s *server) initMiddleware(r chi.Router) error {
	logger := infraMiddleware.NewLog(s.logger)
	writer, err := infraHTTP.NewWriter(s.logger)
	if err != nil {
		return errors.New("error during writer initialization")
	}

	recoverMiddleware, err := infraMiddleware.NewRecover(s.logger, writer)
	if err != nil {
		return errors.New("error during recover middleware initialization")
	}

	r.Use(recoverMiddleware.Handler)
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(s.middleware.timeout))
	r.Use(logger.Handler)

	return nil
}

func (s *server) routerResource(r chi.Router) {
	r.Get("/", s.handler.resource.Index)
	r.Post("/", s.handler.resource.Create)
	r.Get("/{id}", s.handler.resource.Show)
	r.Delete("/{id}", s.handler.resource.Delete)
}

func (s *server) routerSubscription(r chi.Router) {
	r.Get("/", s.handler.subscription.Index)
	r.Post("/", s.handler.subscription.Create)
	r.Get("/{id}", s.handler.subscription.Show)
	r.Delete("/{id}", s.handler.subscription.Delete)
}

func (s *server) routerDocument(r chi.Router) {
	r.Get("/*", s.handler.document.Show)
	r.Put("/*", s.handler.document.Update)
	r.Delete("/*", s.handler.document.Delete)
}

func (s *server) init() error {
	key := "http.server.enable"
	if !s.cfg.GetBool(key) {
		return nil
	}

	s.addr = s.cfg.GetString("http.server.addr")

	timeout, err := s.cfg.GetDuration("http.server.timeout")
	if err != nil {
		return errors.Wrap(err, "error during 'http.server.timeout' parse")
	}
	s.middleware.timeout = timeout

	if s.handler.resource == nil {
		return errors.New("missing handler.resource")
	}

	if s.handler.subscription == nil {
		return errors.New("missing handler.subscription")
	}

	if s.handler.document == nil {
		return errors.New("missing handler.document")
	}

	if s.logger == nil {
		return errors.New("missing logger")
	}
	s.errLogger = level.Error(s.logger)

	s.writeResponse = infraHTTP.WriteResponse(s.logger)

	return s.start()
}
