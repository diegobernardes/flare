package internal

import (
	"net/url"
	"time"
)

// Document represents the resource documents.
type Document struct {
	ID        url.URL
	Revision  int64
	Resource  Resource
	Content   map[string]interface{}
	UpdatedAt time.Time
}
