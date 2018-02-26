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

	"github.com/diegobernardes/flare/domain/consumer/api"
	"github.com/diegobernardes/flare/infra/config"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/pagination"
)

type domain struct {
	logger   log.Logger
	consumer *api.Client
	provider *provider
	cfg      *config.Client
}

func (d *domain) init() error {
	c, err := d.initConsumer()
	if err != nil {
		return err
	}
	d.consumer = c

	return nil
}

func (d *domain) initConsumer() (*api.Client, error) {
	writer, err := infraHTTP.NewWriter(d.logger)
	if err != nil {
		return nil, errors.Wrap(err, "error during http.Writer initialization")
	}

	client := &api.Client{
		Writer:          writer,
		GetID:           func(r *http.Request) string { return chi.URLParam(r, "id") },
		GetURI:          func(id string) string { return fmt.Sprintf("/consumers/%s", id) },
		Repository:      d.provider.cassandraDomainConsumerClient,
		ParsePagination: pagination.Parse(30),
	}
	if err := client.Init(); err != nil {
		return nil, errors.Wrap(err, "error during api initialization")
	}

	return client, nil
}
