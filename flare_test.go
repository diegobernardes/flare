// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"
)

func TestPaginationValid(t *testing.T) {
	tests := []struct {
		name       string
		hasErr     bool
		pagination Pagination
	}{
		{
			"Invalid offset",
			true,
			Pagination{Offset: -1},
		},
		{
			"Invalid offset",
			true,
			Pagination{Limit: 1, Offset: -1},
		},
		{
			"Invalid limit",
			true,
			Pagination{Limit: -1},
		},
		{
			"Invalid limit",
			true,
			Pagination{Offset: 1, Limit: -1},
		},
		{
			"Valid",
			false,
			Pagination{},
		},
		{
			"Valid",
			false,
			Pagination{Limit: 1},
		},
		{
			"Valid",
			false,
			Pagination{Offset: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pagination.Valid()
			if tt.hasErr != (err != nil) {
				t.Errorf("Pagination.valid invalid result, want '%v', got '%v'", tt.hasErr, err)
			}
		})
	}
}
