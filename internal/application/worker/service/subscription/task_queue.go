package subscription

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

// task to create a queue on rabbitmq
// se a fila for assincrona, tem que criar 2 filas

type TaskQueueCreator interface {
	Create(ctx context.Context, subscriptionID string) error
}

type TaskQueue struct {
	Enqueuer     Enqueuer
	QueueCreator TaskQueueCreator
}

func (t TaskQueue) Init() error {
	return nil
}

func (t TaskQueue) Process(ctx context.Context, payload []byte) error {
	subscriptionID, err := t.unmarshal(payload)
	if err != nil {
		return errors.Wrap(err, "unmarshal message")
	}

	if err := t.QueueCreator.Create(ctx, subscriptionID); err != nil {
		return errors.Wrap(err, "create")
	}

	return nil
}

func (t TaskQueue) Consume(ctx context.Context, subscriptionID string) error {
	payload, err := t.marshal(subscriptionID)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	if err := t.Enqueuer.Enqueue(ctx, payload); err != nil {
		return errors.Wrap(err, "enqueue")
	}

	return nil
}

func (t TaskQueue) marshal(subscriptionID string) ([]byte, error) {
	return json.Marshal(map[string]string{"subscriptionID": subscriptionID})
}

func (t TaskQueue) unmarshal(payload []byte) (string, error) {
	result := make(map[string]string)

	if err := json.Unmarshal(payload, &result); err != nil {
		return "", errors.Wrap(err, "unmarshal json")
	}

	subscriptionID, ok := result["subscriptionID"]
	if !ok {
		return "", errors.New("missing subscription id")
	}

	return subscriptionID, nil
}
