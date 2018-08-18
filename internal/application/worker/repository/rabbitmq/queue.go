package rabbitmq

import "context"

type Queue struct{}

func (q Queue) Enqueue(ctx context.Context, payload []byte) error {
	// send the freaking message to rabbitmq
	return nil
}

func (q Queue) Create(ctx context.Context, name, mode string) error {
	// first, check if the queue already exists, if so, don't do nothing.
	return nil
}
