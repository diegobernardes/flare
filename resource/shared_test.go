// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
)

type resourceRepository struct {
	date     time.Time
	base     flare.ResourceRepositorier
	err      error
	createId string
}

func (r *resourceRepository) FindAll(
	ctx context.Context, pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	if r.err != nil {
		return nil, nil, r.err
	}

	resources, page, err := r.base.FindAll(ctx, pagination)
	if err != nil {
		return nil, nil, err
	}

	for i := range resources {
		resources[i].CreatedAt = r.date
	}

	return resources, page, nil
}

func (r *resourceRepository) FindOne(ctx context.Context, id string) (*flare.Resource, error) {
	if r.err != nil {
		return nil, r.err
	}

	res, err := r.base.FindOne(ctx, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = r.date

	return res, nil
}

func (r *resourceRepository) FindByURI(context.Context, string) (*flare.Resource, error) {
	return nil, nil
}

func (r *resourceRepository) Create(ctx context.Context, resource *flare.Resource) error {
	if r.err != nil {
		return r.err
	}
	err := r.base.Create(ctx, resource)
	resource.CreatedAt = r.date
	resource.ID = r.createId
	return err
}

func (r *resourceRepository) Delete(ctx context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	return r.base.Delete(ctx, id)
}

func newResourceRepository(
	date time.Time, resources []flare.Resource, createId string,
) resourceRepository {
	base := memory.NewResource(
		memory.ResourceSubscriptionRepository(memory.NewSubscription()),
	)

	for _, resource := range resources {
		if err := base.Create(context.Background(), &resource); err != nil {
			panic(err)
		}
	}

	return resourceRepository{base: base, date: date, createId: createId}
}

func load(name string) []byte {
	path := fmt.Sprintf("testdata/%s", name)
	f, err := os.Open(path)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during open '%s'", path)))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during read '%s'", path)))
	}
	return content
}

func httpRunner(
	status int,
	header http.Header,
	handler func(w http.ResponseWriter, r *http.Request),
	req *http.Request,
	expectedBody []byte,
) {
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	So(err, ShouldBeNil)
	So(status, ShouldEqual, resp.StatusCode)
	So(header, ShouldResemble, resp.Header)

	if len(body) == 0 && expectedBody == nil {
		return
	}

	b1, b2 := make(map[string]interface{}), make(map[string]interface{})
	err = json.Unmarshal(body, &b1)
	So(err, ShouldBeNil)

	err = json.Unmarshal(expectedBody, &b2)
	So(err, ShouldBeNil)

	So(b1, ShouldResemble, b2)
}
