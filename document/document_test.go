// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestDocumentMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  document
		output string
		hasErr bool
	}{
		{
			"Valid",
			document{
				Id:               "123",
				ChangeFieldValue: "1",
				UpdatedAt:        time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			},
			`{"id":"123","changeFieldValue":"1","updatedAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("document.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			if string(content) != tt.output {
				t.Errorf(
					"document.MarshalJSON invalid result, want '%v', got '%v'", string(content), tt.output,
				)
			}
		})
	}
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
