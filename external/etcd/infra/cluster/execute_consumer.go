package cluster

import (
	"context"
	"errors"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/external/etcd"
	baseCluster "github.com/diegobernardes/flare/infra/cluster"
)

type ExecuteConsumer struct {
	Base *etcd.Consumer
}

func (ec *ExecuteConsumer) Init() error {
	if ec.Base == nil {
		return errors.New("missing Base")
	}
	return nil
}

func (ec *ExecuteConsumer) Load(
	ctx context.Context, nodeID string,
) ([]consumer.Consumer, error) {
	consumers, err := ec.Base.Load(ctx)
	if err != nil {
		return nil, err
	}

	var result []consumer.Consumer
	for _, consumer := range consumers {
		if consumer.NodeID == nodeID {
			c, err := ec.Base.FindByID(ctx, consumer.ID)
			if err != nil {
				return nil, err
			}

			result = append(result, *c)
		}
	}

	return result, nil
}

func (ec *ExecuteConsumer) Watch(
	ctx context.Context, fn func(consumer consumer.Consumer, action string) error, nodeID string,
) context.Context {
	nfn := func(actions ...string) func(baseCluster.Consumer, string) error {
		return func(consumer baseCluster.Consumer, action string) error {
			var found bool
			for _, filter := range actions {
				if filter == action {
					found = true
					break
				}
			}
			if !found {
				return nil
			}

			if consumer.NodeID == nodeID {
				c, err := ec.Base.FindByID(ctx, consumer.ID)
				if err != nil {
					return err
				}

				return fn(*c, action)
			}
			return nil
		}
	}

	assignCtx := ec.Base.WatchAssign(ctx, nfn(
		baseCluster.ActionCreate, baseCluster.ActionDelete, baseCluster.ActionUpdate,
	))
	nctx := ec.Base.Watch(assignCtx, nfn(baseCluster.ActionUpdate))
	return nctx
}
