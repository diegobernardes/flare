package flare

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/resource"
	"github.com/diegobernardes/flare/subscription"
)

type server struct {
	addr       string
	httpServer http.Server
	handler    struct {
		resource     *resource.Service
		subscription *subscription.Service
	}
}

func (s *server) start() {
	s.httpServer = http.Server{
		Addr:    s.addr,
		Handler: s.router(),
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			fmt.Println(errors.Wrap(err, "error during server initialization").Error())

			process, err := os.FindProcess(os.Getpid())
			if err != nil {
				fmt.Println(errors.Wrap(err, "could not find flare process to exit").Error())
				os.Exit(1)
			}
			if err := process.Signal(os.Interrupt); err != nil {
				fmt.Println(errors.Wrap(err, "error during signal send to flare process").Error())
				os.Exit(1)
			}
		}
	}()
}

func (s *server) stop() error {
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		return errors.Wrap(err, "error during server close")
	}
	return nil
}

func (s *server) router() http.Handler {
	r := chi.NewRouter()
	r.Route("/resources", s.routerResource)
	r.Route("/resources/{resourceId}/subscriptions", s.routerSubscription)
	return r
}

func (s *server) routerResource(r chi.Router) {
	r.Get("/", s.handler.resource.HandleIndex)
	r.Post("/", s.handler.resource.HandleCreate)
	r.Get("/{id}", s.handler.resource.HandleShow)
	r.Delete("/{id}", s.handler.resource.HandleDelete)
}

func (s *server) routerSubscription(r chi.Router) {
	r.Get("/", s.handler.subscription.HandleIndex)
	r.Post("/", s.handler.subscription.HandleCreate)
	r.Get("/{id}", s.handler.subscription.HandleShow)
	r.Delete("/{id}", s.handler.subscription.HandleDelete)
}

func newServer(options ...func(*server)) *server {
	s := &server{}

	for _, option := range options {
		option(s)
	}

	if s.addr == "" {
		s.addr = ":8080"
	}

	return s
}

func serverAddr(addr string) func(*server) {
	return func(s *server) { s.addr = addr }
}

func serverHandlerResource(handler *resource.Service) func(*server) {
	return func(s *server) { s.handler.resource = handler }
}

func serverHandlerSubscription(handler *subscription.Service) func(*server) {
	return func(s *server) { s.handler.subscription = handler }
}
