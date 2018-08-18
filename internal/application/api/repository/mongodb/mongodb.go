package mongodb

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type decoder interface {
	Decode(v interface{}) error
}

// timeout returns the lowest value between the parameter and context deadline.
func timeout(ctx context.Context, duration time.Duration) time.Duration {
	ms := duration

	deadline, ok := ctx.Deadline()
	if ok && time.Now().Add(duration).After(deadline) {
		ms = deadline.Sub(time.Now())
	}

	return ms
}

type customError struct {
	cause    error
	notFound bool
}

func (e customError) Error() string { return e.cause.Error() }

func (e customError) NotFound() bool { return e.notFound }

// Timeout  is used to set the timeouts during resource operation.
type Timeout struct {
	Count time.Duration
	Find  time.Duration
}

func (t Timeout) Init() error {
	if t.Count <= 0 {
		return errors.New("invalid Count, expected to be bigger then zero")
	}

	if t.Find <= 0 {
		return errors.New("invalid Find, expected to be bigger then zero")
	}

	return nil
}

type urlView struct {
	Scheme string `bson:"scheme"`
	Host   string `bson:"host"`
	Path   string `bson:"port"`
}
