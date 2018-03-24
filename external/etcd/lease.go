package etcd

// import (
// 	"context"
// 	"sync"

// 	"github.com/coreos/etcd/clientv3"
// 	"github.com/go-kit/kit/log"
// 	"github.com/go-kit/kit/log/level"
// 	"github.com/pkg/errors"
// )

// type Lease struct {
// 	Client     *Client
// 	Logger     log.Logger
// 	TTL        int64
// 	TTLRefresh int

// 	ctx       context.Context
// 	ctxCancel context.Context
// 	id        clientv3.LeaseID
// 	mutex     sync.RWMutex
// }

// func (l *Lease) Init() error {
// 	if l.Client == nil {
// 		return errors.New("missing Client")
// 	}

// 	if l.TTL < 0 {
// 		return errors.New("invalid TTL")
// 	}

// 	if l.TTLRefresh < 0 {
// 		return errors.New("invalid TTLRefresh")
// 	}

// 	if l.TTLRefresh < int(l.TTL) {
// 		return errors.New("invalid TTLRefresh, expected to be bigger or equal TTL")
// 	}

// 	return nil
// }

// func (l *Lease) Start() {
// 	go func() {

// 		resp, err := l.Client.base.Lease.Grant(l.ctx, l.TTL)
// 		if err != nil {
// 			level.Error(l.Logger).Log("message", "error during lease grant", "error", err)
// 		}
// 		l.set(resp.ID)

// 		go func() {
// 			for {
// 				resp, err := l.Client.base.Lease.KeepAliveOnce(l.ctx, resp.ID)
// 				if err != nil {
// 				}

// 			}
// 		}()
// 	}()
// }

// func (l *Lease) Get() clientv3.LeaseID {
// 	l.mutex.RLock()
// 	defer l.mutex.RUnlock()
// 	return l.id
// }

// func (l *Lease) set(id clientv3.LeaseID) {
// 	l.mutex.Lock()
// 	defer l.mutex.Unlock()
// 	l.id = id
// }
