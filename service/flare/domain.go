// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	document "github.com/diegobernardes/flare/domain/document/http"
	resource "github.com/diegobernardes/flare/domain/resource/http"
	subscription "github.com/diegobernardes/flare/domain/subscription/http"
	"github.com/diegobernardes/flare/infra/config"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

type domain struct {
	resource     *resource.Handler
	subscription *subscription.Handler
	document     *document.Handler
	logger       log.Logger
	repository   *repository
	worker       *worker
	cfg          *config.Client
}

func (d *domain) init() error {
	r, err := d.initResource()
	if err != nil {
		return err
	}
	d.resource = r

	s, err := d.initSubscription()
	if err != nil {
		return err
	}
	d.subscription = s

	doc, err := d.initDocument()
	if err != nil {
		return err
	}
	d.document = doc

	return nil
}

func (d *domain) initResource() (*resource.Handler, error) {
	writer, err := infraHTTP.NewWriter(d.logger)
	if err != nil {
		return nil, errors.Wrap(err, "error during http.Writer initialization")
	}

	handler, err := resource.NewHandler(
		resource.HandlerGetResourceID(func(r *http.Request) string { return chi.URLParam(r, "id") }),
		resource.HandlerGetResourceURI(func(id string) string {
			return fmt.Sprintf("/resources/%s", id)
		}),
		resource.HandlerParsePagination(
			infraHTTP.ParsePagination(d.cfg.GetInt("domain.pagination.default-limit")),
		),
		resource.HandlerWriter(writer),
		resource.HandlerRepository(d.repository.base.Resource()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource.Handler initialization")
	}

	return handler, nil
}

func (d *domain) initSubscription() (*subscription.Handler, error) {
	writer, err := infraHTTP.NewWriter(d.logger)
	if err != nil {
		return nil, errors.Wrap(err, "error during http.Writer initialization")
	}

	subscriptionService, err := subscription.NewHandler(
		subscription.HandlerParsePagination(
			infraHTTP.ParsePagination(d.cfg.GetInt("domain.pagination.default-limit")),
		),
		subscription.HandlerWriter(writer),
		subscription.HandlerGetResourceID(func(r *http.Request) string {
			return chi.URLParam(r, "resourceID")
		}),
		subscription.HandlerGetSubscriptionID(func(r *http.Request) string {
			return chi.URLParam(r, "id")
		}),
		subscription.HandlerGetSubscriptionURI(func(resourceId, id string) string {
			return fmt.Sprintf("/resources/%s/subscriptions/%s", resourceId, id)
		}),
		subscription.HandlerResourceRepository(d.repository.base.Resource()),
		subscription.HandlerSubscriptionRepository(d.repository.base.Subscription()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during subscription.Handler initialization")
	}

	return subscriptionService, nil
}

func (d *domain) initDocument() (*document.Handler, error) {
	writer, err := infraHTTP.NewWriter(d.logger)
	if err != nil {
		return nil, errors.Wrap(err, "error during http.Writer initialization")
	}

	documentHandler, err := document.NewHandler(
		document.HandlerDocumentRepository(d.repository.base.Document()),
		document.HandlerGetDocumentID(func(r *http.Request) string { return chi.URLParam(r, "*") }),
		document.HandlerResourceRepository(d.repository.base.Resource()),
		document.HandlerSubscriptionTrigger(d.worker.subscriptionPartition),
		document.HandlerWriter(writer),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during document.Handler initialization")
	}

	return documentHandler, nil
}
