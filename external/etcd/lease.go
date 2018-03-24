package etcd

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type lease struct {
	TTL    time.Duration
	Renew  time.Duration
	Client *Client
}

func (l *lease) new(ctx context.Context) (context.Context, clientv3.LeaseID) {
	nctx, nctxCancel := context.WithCancel(ctx)
	resp, err := l.Client.base.Lease.Grant(ctx, (int64)(l.TTL.Seconds()))
	if err != nil {
		nctxCancel()
		return nctx, 0
	}

	go func() {
		defer nctxCancel()

		for {
			<-time.After(l.Renew)
			if _, err = l.Client.base.KeepAliveOnce(ctx, resp.ID); err != nil {
				return
			}
		}
	}()

	return nctx, resp.ID
}
