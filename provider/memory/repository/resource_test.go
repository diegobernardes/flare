// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"net/url"
	"strconv"
	"testing"

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

				{ID: "1", Endpoint: url.URL{Path: "/product/123/stock/{*}"}},
				{ID: "2", Endpoint: url.URL{Path: "/product/{*}/stock/{*}"}},
				{ID: "3", Endpoint: url.URL{Path: "/product/456/stock/{*}"}},
			},
			5,
			[][]string{
				{"1", "", "product", "123", "stock", "{*}"},
				{"3", "", "product", "456", "stock", "{*}"},
				{"2", "", "product", "{*}", "stock", "{*}"},
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
		c := NewClient()
		r := c.Resource()

		Convey("It should not find a flare.Resource with id 1", func() {
			resource, err := r.FindByID(context.Background(), "1")
			So(resource, ShouldBeNil)
			So(err, ShouldBeError)

			nErr, ok := err.(flare.ResourceRepositoryError)
			So(ok, ShouldBeTrue)
			So(nErr.NotFound(), ShouldBeTrue)
		})

		Convey("When a list of flare.Resource is inserted", func() {
			for i := (int64)(0); i < 10; i++ {
				err := r.Create(context.Background(), &flare.Resource{
					ID:       strconv.FormatInt(i, 10),
					Endpoint: url.URL{Host: strconv.FormatInt(i, 10)},
				})
				So(err, ShouldBeNil)
			}

			Convey("It should find the each flare.Resource by id", func() {
				for i := (int64)(0); i < 10; i++ {
					id := strconv.FormatInt(i, 10)
					resource, err := r.FindByID(context.Background(), id)
					So(resource, ShouldNotBeNil)
					So(err, ShouldBeNil)
					So(resource.ID, ShouldEqual, id)
				}
			})
		})
	})
}

func TestResourceCreate(t *testing.T) {
	Convey("Given a Resource", t, func() {
		c := NewClient()
		r := c.Resource()

		Convey("It should be possible to insert a flare.Resource with id 1", func() {
			err := r.Create(context.Background(), &flare.Resource{ID: "1"})
			So(err, ShouldBeNil)

			Convey("It should not be possible to insert another flare.Resource with id 1", func() {
				err := r.Create(context.Background(), &flare.Resource{ID: "1"})
				So(err, ShouldNotBeNil)

				nErr, ok := err.(flare.ResourceRepositoryError)
				So(ok, ShouldBeTrue)
				So(nErr.AlreadyExists(), ShouldBeTrue)
			})
		})

		Convey("It should be possible to insert a flare.Resource with app.com address", func() {
			err := r.Create(context.Background(), &flare.Resource{
				ID:       "1",
				Endpoint: url.URL{Scheme: "http", Host: "app.com"},
			})
			So(err, ShouldBeNil)

			msg := "It should not be possible to insert another flare.Resource at the same address"
			Convey(msg, func() {
				err := r.Create(context.Background(), &flare.Resource{
					ID:       "2",
					Endpoint: url.URL{Scheme: "http", Host: "app.com"},
				})
				So(err, ShouldNotBeNil)

				nErr, ok := err.(flare.ResourceRepositoryError)
				So(ok, ShouldBeTrue)
				So(nErr.AlreadyExists(), ShouldBeTrue)
			})
		})
	})
}

func TestResourceDelete(t *testing.T) {
	Convey("Given a Resource", t, func() {
		c := NewClient()
		r := c.Resource()

		Convey("It should not be possible to delete a flare.Resource that does not exist", func() {
			err := r.Delete(context.Background(), "1")
			So(err, ShouldNotBeNil)

			nErr, ok := err.(flare.ResourceRepositoryError)
			So(ok, ShouldBeTrue)
			So(nErr.NotFound(), ShouldBeTrue)
		})
	})
}
