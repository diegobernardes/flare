package subscription

import (
	"context"

	"github.com/pkg/errors"
)

type QueueCreator interface {
	Create(ctx context.Context, subscriptionID, mode string) error
}

type QueueSubscriptionMode interface {
	Mode(ctx context.Context, subscriptionID string) (string, error)
}

type Queue struct {
	Creator          QueueCreator
	SubscriptionMode QueueSubscriptionMode
}

func (q Queue) Create(ctx context.Context, subscriptionID string) error {
	mode, err := q.SubscriptionMode.Mode(ctx, subscriptionID)
	if err != nil {
		return errors.Wrapf(err, "error while discovering subscription '%s' mode", subscriptionID)
	}

	if err := q.Creator.Create(ctx, subscriptionID, mode); err != nil {
		return errors.Wrap(err, "error during queue create")
	}

	return nil
}
