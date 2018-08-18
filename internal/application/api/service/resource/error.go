package resource

import "github.com/pkg/errors"

const (
	errorKindClient        = "client"
	errorKindServer        = "server"
	errorKindNotFound      = "notFound"
	errorKindAlreadyExists = "alreadyExists"
)

type Error struct {
	Cause   error
	Message string

	kind string
}

func (e Error) Error() string {
	return errors.Wrap(e.Cause, e.Message).Error()
}

func (e Error) Client() bool { return e.kind == errorKindClient }

func (e Error) Server() bool { return e.kind == errorKindServer }

func (e Error) NotFound() bool { return e.kind == errorKindNotFound }

func (e Error) AlreadyExists() bool { return e.kind == errorKindAlreadyExists }
