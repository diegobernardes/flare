package internal

import (
	"net/url"
)

// Resource represents the apis Flare track and the info to detect changes on documents.
type Resource struct {
	ID       string
	Endpoint url.URL
	Change   ResourceChange
}

// ResourceChange holds the information to detect document change.
type ResourceChange struct {
	Field  string
	Format string
}
