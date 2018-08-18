package http

import (
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCheckPermitedQS(t *testing.T) {
	Convey("Feature: Check permited query on url", t, func() {
		Convey("Given a list of urls", func() {
			tests := []struct {
				title       string
				query       url.Values
				permited    []string
				shouldError bool
			}{
				{
					"have a error because of a invalid query #1",
					url.Values{"key": []string{"value"}},
					[]string{"limit", "offset"},
					true,
				},
				{
					"have a error because of a invalid query #2",
					url.Values{"key": []string{"value"}, "limit": []string{"30"}},
					[]string{"limit", "offset"},
					true,
				},
				{
					"success #1",
					url.Values{},
					[]string{},
					false,
				},
				{
					"success #2",
					url.Values{},
					[]string{"limit"},
					false,
				},
				{
					"success #3",
					url.Values{"limit": []string{"30"}},
					[]string{"limit"},
					false,
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					err := CheckPermitedQS(tt.query, tt.permited)
					So(err != nil, ShouldEqual, tt.shouldError)
				})
			}
		})
	})
}
