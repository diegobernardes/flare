package document

import (
	"encoding/json"
	"time"

	"github.com/diegobernardes/flare"
)

type document struct {
	base *flare.Document
}

func (d *document) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id               string      `json:"id"`
		ChangeFieldValue interface{} `json:"changeFieldValue"`
		UpdatedAt        string      `json:"updatedAt"`
	}{
		Id:               d.base.Id,
		ChangeFieldValue: d.base.ChangeFieldValue,
		UpdatedAt:        d.base.UpdatedAt.Format(time.RFC3339),
	})
}

func transformDocument(d *flare.Document) *document {
	return &document{d}
}

type response struct {
	Error    *responseError `json:"error,omitempty"`
	Document *document      `json:"document,omitempty"`
}

type responseError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}
