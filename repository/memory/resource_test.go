// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memory

import (
	"context"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
)

func TestResourceGenResourceSegments(t *testing.T) {
	tests := []struct {
		expect      string
		result      string
		resources   []flare.Resource
		qtySegments int
		want        [][]string
	}{
		{
			"When the list is nil",
			"The result should be a empty list of flare.Resource",
			nil,
			0,
			[][]string{},
		},
		{
			"When the list is not nil",
			`The result should contain a list of list of strings with the flare.Resource id and each
path segment`,
			[]flare.Resource{
				{Id: "1", Path: "/product/123/stock/{track}"},
				{Id: "2", Path: "/product/{*}/stock/{track}"},
				{Id: "3", Path: "/product/456/stock/{track}"},
			},
			5,
			[][]string{
				{"1", "", "product", "123", "stock", "{track}"},
				{"3", "", "product", "456", "stock", "{track}"},
				{"2", "", "product", "{*}", "stock", "{track}"},
			},
		},
	}

	Convey("Given a list of flare.Resource", t, func() {
		for _, tt := range tests {
			Convey(tt.expect, func() {
				var r Resource
				result := r.genResourceSegments(tt.resources, tt.qtySegments)

				Convey(tt.result, func() { So(result, ShouldResemble, tt.want) })
			})
		}
	})
}

func TestResourceFindOne(t *testing.T) {
	Convey("Given a Resource", t, func() {
		r := NewResource()

		Convey("It should not find a flare.Resource with id 1", func() {
			resource, err := r.FindOne(context.Background(), "1")
			So(resource, ShouldBeNil)
			So(err, ShouldBeError)

			nErr, ok := err.(flare.ResourceRepositoryError)
			So(ok, ShouldBeTrue)
			So(nErr.NotFound(), ShouldBeTrue)
		})

		Convey("When a list of flare.Resource is inserted", func() {
			for i := (int64)(0); i < 10; i++ {
				r.Create(context.Background(), &flare.Resource{Id: strconv.FormatInt(i, 10)})
			}

			Convey("It should find the each flare.Resource by id", func() {
				for i := (int64)(0); i < 10; i++ {
					id := strconv.FormatInt(i, 10)
					resource, err := r.FindOne(context.Background(), id)
					So(resource, ShouldNotBeNil)
					So(err, ShouldBeNil)
					So(resource.Id, ShouldEqual, id)
				}
			})
		})
	})
}

func TestResourceCreate(t *testing.T) {
	Convey("Given a Resource", t, func() {
		r := NewResource()

		Convey("It should be possible to insert a flare.Resource with id 1", func() {
			err := r.Create(context.Background(), &flare.Resource{Id: "1"})
			So(err, ShouldBeNil)

			Convey("It should not be possible to insert another flare.Resource with id 1", func() {
				err := r.Create(context.Background(), &flare.Resource{Id: "1"})
				So(err, ShouldNotBeNil)

				nErr, ok := err.(flare.ResourceRepositoryError)
				So(ok, ShouldBeTrue)
				So(nErr.AlreadyExists(), ShouldBeTrue)
			})
		})

		Convey("It should be possible to insert a flare.Resource with app.com address", func() {
			err := r.Create(context.Background(), &flare.Resource{
				Id:        "1",
				Addresses: []string{"http://app.com"},
			})
			So(err, ShouldBeNil)

			msg := "It should not be possible to insert another flare.Resource at the same address"
			Convey(msg, func() {
				err := r.Create(context.Background(), &flare.Resource{
					Id:        "2",
					Addresses: []string{"http://app.com"},
				})
				So(err, ShouldNotBeNil)

				nErr, ok := err.(flare.ResourceRepositoryError)
				So(ok, ShouldBeTrue)
				So(nErr.PathConflict(), ShouldBeTrue)
			})
		})
	})
}

func TestResourceDelete(t *testing.T) {
	Convey("Given a Resource", t, func() {
		r := NewResource()

		Convey("It should not be possible to delete a flare.Resource that does not exist", func() {
			err := r.Delete(context.Background(), "1")
			So(err, ShouldNotBeNil)

			nErr, ok := err.(flare.ResourceRepositoryError)
			So(ok, ShouldBeTrue)
			So(nErr.NotFound(), ShouldBeTrue)
		})
	})

	Convey("Given a Resource with a mocked flare.SubscriptionRepositorier", t, func() {
		sr := &subscriptionRepositorier{
			deleteErr: errors.New("error during delete"),
			base:      NewSubscription(),
		}
		r := NewResource(ResourceSubscriptionRepository(sr))

		Convey("It is expected to have a error", func() {
			err := r.Delete(context.Background(), "1")
			So(err, ShouldNotBeNil)

			// nErr, ok := err.(flare.ResourceRepositoryError)
			// So(ok, ShouldBeTrue)
			// So(nErr.NotFound(), ShouldBeTrue)
		})
	})
}

type subscriptionRepositorier struct {
	flare.SubscriptionRepositorier
	base      flare.SubscriptionRepositorier
	deleteErr error
}

func (r *subscriptionRepositorier) Delete(ctx context.Context, resourceId string, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.base.Delete(ctx, resourceId, id)
}

func (r *subscriptionRepositorier) FindAll(
	ctx context.Context, pagination *flare.Pagination, id string,
) ([]flare.Subscription, *flare.Pagination, error) {
	return r.base.FindAll(ctx, pagination, id)
}
