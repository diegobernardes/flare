package etcd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/cluster"
)

// Node is used to register the node at the cluster.
type Node struct {
	ID       string
	LeaseTTL time.Duration
	Client   *Client
	Logger   log.Logger
}

// Init check if all the requirements are fulfilled.
func (n *Node) Init() error {
	if n.ID == "" {
		return errors.New("invalid ID")
	}

	if n.LeaseTTL < 0 {
		return errors.New("invalid LeaseTTL")
	}

	if n.Client == nil {
		return errors.New("missing Client")
	}

	return nil
}

// Join is used to register the node in the cluster. The return is a context that should be used as
// a flag that indicates the register status. If the context get canceled, the node is no longer
// registered. These are the cancellation scenarios:
//
//   - ctx param get canceled
//   - node ctx get canceled
//   - error during join
//
func (n *Node) Join(ctx context.Context) context.Context {
	nctx, nctxCancel := context.WithCancel(ctx)
	lease, err := n.Client.base.Lease.Grant(nctx, (int64)(n.LeaseTTL.Seconds()))
	if err != nil {
		nctxCancel()
		return nctx
	}

	keepAliveLease, err := n.Client.base.Lease.KeepAlive(nctx, lease.ID)
	if err != nil {
		nctxCancel()
		return nctx
	}

	kv := clientv3.NewKV(n.Client.base)
	key := fmt.Sprintf("/node/%s", n.ID)
	leaseID := strconv.FormatInt((int64)(lease.ID), 16)

	watch := n.Client.base.Watch(nctx, key)
	if _, err := kv.Put(nctx, key, leaseID, clientv3.WithLease(lease.ID)); err != nil {
		nctxCancel()
		return nctx
	}

	go func() {
		defer nctxCancel()

		for {
			select {
			case we := <-watch:
				for _, event := range we.Events {
					if event.Type == mvccpb.DELETE {
						return
					}
				}
			case keepAlive := <-keepAliveLease:
				if keepAlive == nil {
					return
				}
			}
		}
	}()

	return nctx
}

// Load all the nodes at the cluster.
func (n *Node) Load(ctx context.Context) ([]string, error) {
	kv := clientv3.NewKV(n.Client.base)

	resp, err := kv.Get(ctx, "/node/", clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, errors.Wrap(err, "error during fetch nodes")
	}

	var ids []string
	for _, value := range resp.Kvs {
		id := strings.Split(string(value.Key), "/")[2]
		ids = append(ids, id)
	}

	return ids, nil
}

// Watch for node changes.
func (n *Node) Watch(ctx context.Context, fn func(nodeID, action string) error) context.Context {
	watch := clientv3.NewWatcher(n.Client.base)

	ch := watch.Watch(ctx, "/node", clientv3.WithPrefix(), clientv3.WithKeysOnly())
	ctx, ctxCancel := context.WithCancel(ctx)

	go func() {
		defer ctxCancel()

		for change := range ch {
			if change.Canceled {
				return
			}

			for _, event := range change.Events {
				action := cluster.ActionCreate
				if event.Type == mvccpb.DELETE {
					action = cluster.ActionDelete
				}

				id := strings.Split(string(event.Kv.Key), "/")[2]
				if err := fn(id, action); err != nil {
					return
				}
			}
		}
	}()

	return ctx
}

func (n *Node) lease(ctx context.Context, id string) (clientv3.LeaseID, error) {
	kv := clientv3.NewKV(n.Client.base)
	resp, err := kv.Get(ctx, fmt.Sprintf("/node/%s", id))
	if err != nil {
		return 0, errors.Wrapf(err, "error during fetch lease from node '%s'", id)
	}

	if len(resp.Kvs) == 0 {
		return 0, nil
	}

	value, err := strconv.ParseInt(string(resp.Kvs[0].Value), 16, 64)
	if err != nil {
		return 0, errors.Wrap(err, "error during parse lease value")
	}

	return clientv3.LeaseID(value), nil
}
