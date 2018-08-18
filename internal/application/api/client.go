package api

import (
	"context"
	"fmt"
	coreHTTP "net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-kit/kit/log"
	"github.com/kr/pretty"

	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
	"github.com/diegobernardes/flare/internal/application/api/repository/mongodb"
	"github.com/diegobernardes/flare/internal/application/api/service/resource"
	"github.com/diegobernardes/flare/internal/application/api/service/resource/http"
	"github.com/diegobernardes/flare/internal/application/api/service/subscription"
	shttp "github.com/diegobernardes/flare/internal/application/api/service/subscription/http"
)

// Variables set with ldflags during compilation.
var (
	Version   = ""
	BuildTime = ""
	Commit    = ""
	GoVersion = runtime.Version()
)

type Client struct {
	Config string
}

func (c Client) Init() error {
	return nil
}

func (c Client) Start() error {
	mongoClient := mongodb.Client{
		ConnectionString: "mongodb://localhost:27017/flare",
	}
	if err := mongoClient.Init(); err != nil {
		panic(err)
	}

	mongoResource := mongodb.Resource{
		Database: mongoClient.Database("flare"),
		Timeout: mongodb.Timeout{
			Count: 100 * time.Millisecond,
			Find:  100 * time.Millisecond,
		},
	}
	if err := mongoResource.Init(); err != nil {
		panic(err)
	}

	mongoSubscription := mongodb.Subscription{
		Database: mongoClient.Database("flare"),
		Timeout: mongodb.Timeout{
			Count: 100 * time.Millisecond,
			Find:  100 * time.Millisecond,
		},
	}
	if err := mongoSubscription.Init(); err != nil {
		panic(err)
	}

	if err := mongoResource.EnsureIndex(context.Background()); err != nil {
		pretty.Println(err)
		panic(err)
	}

	resourceService := resource.Service{
		Repository: mongoResource,
	}
	if err := resourceService.Init(); err != nil {
		panic(err)
	}

	subscriptionService := subscription.Service{
		Repository: mongoSubscription,
	}
	if err := subscriptionService.Init(); err != nil {
		panic(err)
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	writer, err := infraHTTP.NewWriter(logger)
	if err != nil {
		panic(err)
	}

	handler := http.Handler{
		Service:         resourceService,
		ParsePagination: infraHTTP.ParsePagination(30),
		Writer:          writer,
		ExtractID:       func(r *coreHTTP.Request) string { return chi.URLParam(r, "id") },
		GenURI:          func(id string) string { return fmt.Sprintf("/resources/%s", id) },
	}
	if err := handler.Init(); err != nil {
		panic(err)
	}

	handlerSubscription := shttp.Handler{
		Service:               subscriptionService,
		ParsePagination:       infraHTTP.ParsePagination(30),
		Writer:                *writer,
		ExtractResourceID:     func(r *coreHTTP.Request) string { return chi.URLParam(r, "resourceID") },
		ExtractSubscriptionID: func(r *coreHTTP.Request) string { return chi.URLParam(r, "id") },
		GenURI:                func(id string) string { return fmt.Sprintf("/resources/%s", id) },
	}
	if err := handlerSubscription.Init(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Get("/resources", handler.Index)
	r.Get("/resources/{id}", handler.Show)
	r.Post("/resources", handler.Create)
	r.Delete("/resources/{id}", handler.Delete)
	r.Get("/resources/{resourceID}/subscriptions", handlerSubscription.Index)
	r.Get("/resources/{resourceID}/subscriptions/{id}", handlerSubscription.Show)
	coreHTTP.ListenAndServe(":8080", r)

	return nil
}

func (c Client) Stop() error {
	return nil
}

func (c Client) Setup(context.Context) error {
	return nil
}
