// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	memory "github.com/diegobernardes/flare/provider/memory/repository"
)

func TestRun(t *testing.T) {
	fn, ip, err := initMongoDBContainer()
	if err != nil {
		panic(err)
	}
	defer fn()

	repository, stop, err := initMongoDBResourceRepositoriy(ip)
	if err != nil {
		panic(err)
	}
	defer stop()

	client := memory.NewClient()

	t.Run("MongoDB", testResourceRun(repository))
	t.Run("Memory", testResourceRun(client.Resource()))
}

func testResourceRun(repo flare.ResourceRepositorier) func(*testing.T) {
	return func(t *testing.T) {
		Convey("Given a resource repository", t, func() {
			Convey("Before insert", func() {
				testResourceFindOneWithError(repo)
				testResourceFindAllPreFeed(repo)
				testResourceFindByURIPreFeed(repo)
			})

			Convey("During insert", func() {
				testResourceCreate(repo)
				testResourceFindByURI(repo)
			})

			Convey("During delete", func() {
				testResourceDelete(repo)
				testResourceFindOneWithError(repo)
				testResourceFindAllPreFeed(repo)
			})

			Convey("Before delete", func() {
				testResourceFindOneWithError(repo)
				testResourceFindAllPreFeed(repo)
				testResourceFindByURIPreFeed(repo)
			})
		})
	}
}

func testResourceFindByURIPreFeed(repo flare.ResourceRepositorier) {
	Convey("It should return error during FindByURI", func() {
		resource, err := repo.FindByURI(context.Background(), "http://app.com/users/123")
		So(err, ShouldNotBeNil)
		So(resource, ShouldBeNil)
	})
}

func testResourceFindByURI(repo flare.ResourceRepositorier) {
	Convey("It should return a resource by uri", func() {
		resource, err := repo.FindByURI(context.Background(), "http://app.com/users/123")
		So(err, ShouldBeNil)
		So(resource, ShouldNotBeNil)
	})
}

func testResourceFindAllPreFeed(repo flare.ResourceRepositorier) {
	Convey("It should return a empty list of resources", func() {
		resource, pagination, err := repo.Find(context.Background(), &flare.Pagination{Limit: 10})
		So(err, ShouldBeNil)
		So(pagination, ShouldResemble, &flare.Pagination{Limit: 10})
		So(resource, ShouldResemble, []flare.Resource{})
	})
}

func testResourceFindOneWithError(repo flare.ResourceRepositorier) {
	Convey("It should not be possible to get a resource", func() {
		resource, err := repo.FindByID(context.Background(), "1")
		So(resource, ShouldBeNil)
		So(err, ShouldNotBeNil)

		nErr, ok := err.(flare.ResourceRepositoryError)
		So(ok, ShouldBeTrue)
		So(nErr.NotFound(), ShouldBeTrue)
	})
}

func testResourceCreate(repo flare.ResourceRepositorier) {
	Convey("It should be possible to insert a resource", func() {
		err := repo.Create(context.Background(), &flare.Resource{
			ID:        "1",
			Addresses: []string{"http://app.com"},
			Path:      "/users/{*}",
		})
		So(err, ShouldBeNil)
	})

	Convey("It should not be possible to insert another resource with same id", func() {
		err := repo.Create(context.Background(), &flare.Resource{
			ID:        "1",
			Addresses: []string{"http://app.com"},
			Path:      "/sample/{*}",
		})
		So(err, ShouldNotBeNil)

		nErr, ok := err.(flare.ResourceRepositoryError)
		So(ok, ShouldBeTrue)
		So(nErr.AlreadyExists(), ShouldBeTrue)
	})

	for i, wildcard := range []string{"id", "*"} {
		msg := fmt.Sprintf(
			"It should not be possible to insert a resource with the same address %d", i,
		)
		Convey(msg, func() {
			err := repo.Create(context.Background(), &flare.Resource{
				ID:        "2",
				Addresses: []string{"http://app.com"},
				Path:      fmt.Sprintf("/users/{%s}", wildcard),
			})
			So(err, ShouldNotBeNil)

			nErr, ok := err.(flare.ResourceRepositoryError)
			So(ok, ShouldBeTrue)
			So(nErr.AlreadyExists(), ShouldBeTrue)
		})
	}
}

func testResourceDelete(repo flare.ResourceRepositorier) {
	Convey("It should be possible to delete a resource", func() {
		err := repo.Delete(context.Background(), "1")
		So(err, ShouldBeNil)
	})

	Convey("It should not be possible to delete another resource with same id", func() {
		err := repo.Delete(context.Background(), "1")
		So(err, ShouldNotBeNil)

		nErr, ok := err.(flare.ResourceRepositoryError)
		So(ok, ShouldBeTrue)
		So(nErr.NotFound(), ShouldBeTrue)
	})
}
