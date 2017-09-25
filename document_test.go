// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"
)

// Quebrar esse teste nos testes menores, ta mt rande.... dificil de entender.
// mudar tb os valores pra nao precisar ficar passando os 2 sempre, o doc ja tem acesso...

func TestDocumentValid(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		doc    Document
	}{
		{
			"Invalid Id",
			true,
			Document{},
		},
		{
			"Invalid ChangeFieldValue",
			true,
			Document{Id: "1"},
		},
		{
			"Invalid Resource.Change",
			true,
			Document{
				Id:               "1",
				ChangeFieldValue: 1,
				Resource:         Resource{Change: ResourceChange{}},
			},
		},
		{
			"Invalid ChangeFieldValue",
			true,
			Document{
				Id:               "1",
				ChangeFieldValue: 1,
				Resource: Resource{
					Change: ResourceChange{
						Field: "revision",
						Kind:  ResourceChangeString,
					},
				},
			},
		},
		{
			"Invalid ChangeFieldValue",
			true,
			Document{
				Id:               "1",
				ChangeFieldValue: 1,
				Resource: Resource{
					Change: ResourceChange{
						Field:      "revision",
						Kind:       ResourceChangeDate,
						DateFormat: "2006-01-02",
					},
				},
			},
		},
		{
			"Invalid date on ChangeFieldValue",
			true,
			Document{
				Id:               "1",
				ChangeFieldValue: "sample",
				Resource: Resource{
					Change: ResourceChange{
						Field:      "revision",
						Kind:       ResourceChangeDate,
						DateFormat: "2006-01-02",
					},
				},
			},
		},
		{
			"Valid",
			false,
			Document{
				Id:               "1",
				ChangeFieldValue: 1,
				Resource: Resource{
					Change: ResourceChange{Field: "revision", Kind: ResourceChangeInteger},
				},
			},
		},
		{
			"Valid",
			false,
			Document{
				Id:               "1",
				ChangeFieldValue: "2006-01-02",
				Resource: Resource{
					Change: ResourceChange{
						Field:      "revision",
						Kind:       ResourceChangeDate,
						DateFormat: "2006-01-02",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Valid()
			if tt.hasErr != (err != nil) {
				t.Errorf("Document.valid invalid result, want '%v', got '%v'", tt.hasErr, err)
			}
		})
	}
}

func TestDocumentNewer(t *testing.T) {
	tests := []struct {
		name      string
		reference *Document
		target    Document
		hasErr    bool
		newer     bool
	}{
		{
			"Invalid",
			&Document{},
			Document{},
			true,
			false,
		},
		{
			"Invalid",
			&Document{},
			Document{
				Resource: Resource{
					Change: ResourceChange{
						Kind: "sample",
					},
				},
			},
			true,
			false,
		},
		{
			"Invalid",
			&Document{},
			Document{
				Resource: Resource{
					Change: ResourceChange{
						Kind: ResourceChangeDate,
					},
				},
			},
			true,
			false,
		},
		{
			"Invalid",
			&Document{},
			Document{
				Resource: Resource{
					Change: ResourceChange{
						Kind: ResourceChangeInteger,
					},
				},
			},
			true,
			false,
		},
		{
			"Invalid",
			&Document{},
			Document{
				Resource: Resource{
					Change: ResourceChange{
						Kind: ResourceChangeString,
					},
				},
			},
			true,
			false,
		},
		{
			"Valid",
			nil,
			Document{},
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newer, err := tt.target.Newer(tt.reference)
			if tt.hasErr != (err != nil) {
				t.Errorf("Document.Newer error, want '%v', got '%v'", tt.hasErr, err)
				t.FailNow()
			}

			if tt.newer != newer {
				t.Errorf("Document.Newer invalid result, want '%v', got '%v'", tt.newer, newer)
			}
		})
	}
}

func TestDocumentNewerDate(t *testing.T) {
	tests := []struct {
		name     string
		document Document
		value    interface{}
		hasErr   bool
		newer    bool
	}{
		{
			"Invalid",
			Document{},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: "2006-01-02"},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: "2006-01-02"},
			"2006-01-02",
			true,
			false,
		},
		{
			"Invalid",
			Document{
				ChangeFieldValue: "2006-01-02",
				Resource: Resource{
					Change: ResourceChange{
						DateFormat: "2006-01-02",
					},
				},
			},
			"",
			true,
			false,
		},
		{
			"Valid",
			Document{
				ChangeFieldValue: "2006-01-02",
				Resource: Resource{
					Change: ResourceChange{
						DateFormat: "2006-01-02",
					},
				},
			},
			"2006-01-02",
			false,
			false,
		},
		{
			"Valid",
			Document{
				ChangeFieldValue: "2007-01-02",
				Resource: Resource{
					Change: ResourceChange{
						DateFormat: "2006-01-02",
					},
				},
			},
			"2006-01-02",
			false,
			true,
		},
		{
			"Valid",
			Document{
				ChangeFieldValue: "2005-01-02",
				Resource: Resource{
					Change: ResourceChange{
						DateFormat: "2006-01-02",
					},
				},
			},
			"2006-01-02",
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newer, err := tt.document.newerDate(tt.value)
			if tt.hasErr != (err != nil) {
				t.Errorf("Document.newerDate error, want '%v', got '%v'", tt.hasErr, err)
				t.FailNow()
			}

			if tt.newer != newer {
				t.Errorf("Document.newerDate invalid result, want '%v', got '%v'", tt.newer, newer)
			}
		})
	}
}

func TestDocumentNewerInteger(t *testing.T) {
	tests := []struct {
		name     string
		document Document
		value    interface{}
		hasErr   bool
		newer    bool
	}{
		{
			"Invalid",
			Document{ChangeFieldValue: 1},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: float64(1)},
			"sample",
			true,
			false,
		},
		{
			"Valid",
			Document{ChangeFieldValue: float64(1)},
			float64(1),
			false,
			false,
		},
		{
			"Valid",
			Document{ChangeFieldValue: float64(2)},
			float64(1),
			false,
			true,
		},
		{
			"Valid",
			Document{ChangeFieldValue: float64(0)},
			float64(1),
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newer, err := tt.document.newerInteger(tt.value)
			if tt.hasErr != (err != nil) {
				t.Errorf("Document.newerInteger error, want '%v', got '%v'", tt.hasErr, err)
				t.FailNow()
			}

			if tt.newer != newer {
				t.Errorf("Document.newerInteger invalid result, want '%v', got '%v'", tt.newer, newer)
			}
		})
	}
}

func TestDocumentNewerString(t *testing.T) {
	tests := []struct {
		name     string
		document Document
		value    interface{}
		hasErr   bool
		newer    bool
	}{
		{
			"Invalid",
			Document{},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: 1},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: "sample"},
			nil,
			true,
			false,
		},
		{
			"Invalid",
			Document{ChangeFieldValue: "sample"},
			1,
			true,
			false,
		},
		{
			"Valid",
			Document{ChangeFieldValue: "sample"},
			"sample",
			false,
			false,
		},
		{
			"Valid",
			Document{ChangeFieldValue: "sample1"},
			"sample0",
			false,
			true,
		},
		{
			"Valid",
			Document{ChangeFieldValue: "sample0"},
			"sample1",
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newer, err := tt.document.newerString(tt.value)
			if tt.hasErr != (err != nil) {
				t.Errorf("Document.newerString error, want '%v', got '%v'", tt.hasErr, err)
				t.FailNow()
			}

			if tt.newer != newer {
				t.Errorf("Document.newerString invalid result, want '%v', got '%v'", tt.newer, newer)
			}
		})
	}
}
