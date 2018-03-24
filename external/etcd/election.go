package etcd

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

// Election is used to choose a master on cluster.
type Election struct {
	Client *Client
	Logger log.Logger
	Node   *Node
	NodeID string
	TTL    int
}

// Init check if all the requirements are fulfilled.
func (e *Election) Init() error {
	if e.Client == nil {
		return errors.New("missing Client")
	}

	if e.Logger == nil {
		return errors.New("missing Logger")
	}

	if e.Node == nil {
		return errors.New("missing Node")
	}

	if e.NodeID == "" {
		return errors.New("invalid NodeID")
	}

	if e.TTL < 0 {
		return errors.New("invalid TTL")
	}

	return nil
}

// Elect a new master on cluster. The return is a context that should be used as a flag that
// indicates the election status. If the context get canceled, the node is no longer registred.
// These are the cancellation scenarios:
//
//   - ctx param get canceled
//   - election ctx get canceled
//   - error during election
//
func (e *Election) Elect(ctx context.Context) context.Context {
	nctx, nctxCancel := context.WithCancel(ctx)
	session, err := concurrency.NewSession(
		e.Client.base,
		concurrency.WithTTL(5),
		concurrency.WithContext(nctx),
	)
	if err != nil {
		nctxCancel()
		return nctx
	}

	keepAliveLease, err := e.Client.base.Lease.KeepAlive(nctx, session.Lease())
	if err != nil {
		nctxCancel()
		return nctx
	}

	election := concurrency.NewElection(session, "/election")
	if err = election.Campaign(nctx, e.NodeID); err != nil {
		nctxCancel()
		return nctx
	}
	watch := e.Client.base.Watch(nctx, election.Key())

	go func() {
		defer nctxCancel()

		for {
			select {
			case keepAlive := <-keepAliveLease:
				if keepAlive == nil {
					return
				}
			case _, ok := <-election.Observe(nctx):
				if !ok {
					return
				}
				<-time.After(1 * time.Second)
			case we := <-watch:
				for _, event := range we.Events {
					if event.Type == mvccpb.DELETE {
						return
					}
				}
			case <-time.After(time.Second):
				return
			}
		}
	}()

	return nctx
}
