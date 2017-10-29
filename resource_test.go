// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"reflect"
	"testing"
)

func TestResourceChangeValid(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		rc     ResourceChange
	}{
		{
			"Missing field",
			true,
			ResourceChange{},
		},
		{
			"Missing kind",
			true,
			ResourceChange{Field: "updatedAt"},
		},
		{
			"Missing dateFormat",
			true,
			ResourceChange{Field: "updatedAt", Kind: ResourceChangeDate},
		},
		{
			"Valid",
			false,
			ResourceChange{Field: "updatedAt", Kind: ResourceChangeDate, DateFormat: "2006-01-02"},
		},
		{
			"Valid",
			false,
			ResourceChange{Field: "revision", Kind: ResourceChangeInteger},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rc.Valid()
			if tt.hasErr != (err != nil) {
				t.Errorf("ResourceChange.valid invalid result, want '%v', got '%v'", tt.hasErr, err)
			}
		})
	}
}

func TestResourceWildcardReplace(t *testing.T) {
	tests := []struct {
		name       string
		resource   Resource
		id         string
		rawContent []string
		expected   []string
		hasErr     bool
	}{
		{
			"Valid",
			Resource{Path: "/resource/{id}"},
			"/resource/123",
			[]string{"{id}", `{"id":"{id}"}`},
			[]string{"123", `{"id":"123"}`},
			false,
		},
		{
			"Invalid",
			Resource{},
			"%zzzzz",
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := tt.resource.WildcardReplace(tt.id)
			if tt.hasErr != (err != nil) {
				t.Errorf("Resource.WildcardReplace invalid result, want '%v', got '%v'", tt.hasErr, err)
			}

			for i, value := range tt.rawContent {
				tt.rawContent[i] = fn(value)
			}

			if !reflect.DeepEqual(tt.rawContent, tt.expected) {
				t.Errorf(
					"ResourceChange.WildcardReplace invalid result, want '%v', got '%v'",
					tt.expected,
					tt.rawContent,
				)
			}
		})
	}
}
