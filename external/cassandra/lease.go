package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gocql/gocql"
	"github.com/pkg/errors"
)

const (
	leaseLock     = "lock"
	leaseRegistry = "registry"
	leaseAssign   = "assign"
)

// Lease implements a distributed lock.
type Lease struct {
	Client        *Client
	Logger        log.Logger
	TTL           time.Duration
	Renew         time.Duration
	DeleteTimeout time.Duration

	nextRun *time.Time
}

// Init validate if the parameters are valid before start the process.
func (l *Lease) Init() error {
	if l.Client == nil {
		return errors.New("missing Client")
	}

	if l.Logger == nil {
		return errors.New("missing Logger")
	}

	if l.TTL <= 0 {
		return errors.New("invalid TTL")
	}

	if l.Renew <= 0 {
		return errors.New("invalid Renew")
	}

	if l.DeleteTimeout <= 0 {
		return errors.New("insvalid DeleteTimeout")
	}

	return nil
}

// Lock a given key for a node.
func (l *Lease) Lock(ctx context.Context, key, nodeID string) context.Context {
	nctx, nctxCancel := context.WithCancel(ctx)

	l.delay(nctx, func() {
		if !l.exec(nctx, leaseLock, key, nodeID, false) {
			nctxCancel()
			return
		}

		go l.keepalive(nctx, nctxCancel, leaseLock, key, nodeID, true)
	})

	return nctx
}

// Join the node into cluster.
func (l *Lease) Join(ctx context.Context, nodeID string) context.Context {
	nctx, nctxCancel := context.WithCancel(ctx)

	l.delay(nctx, func() {
		key := fmt.Sprintf("/node/%s", nodeID)

		if !l.exec(nctx, leaseRegistry, key, nodeID, false) {
			nctxCancel()
			return
		}

		go l.keepalive(nctx, nctxCancel, leaseRegistry, key, nodeID, true)
	})

	return nctx
}

func (l *Lease) exec(ctx context.Context, kind, key, nodeID string, exists bool) bool {
	seconds := func() int { return (int)(l.TTL.Seconds()) }

start:
	var query *gocql.Query

	if exists {
		query = l.Client.Session.Query(
			`UPDATE leases USING TTL ?
					SET node_id = ?, updated_at = ?, type = ?
			  WHERE key = ? IF node_id = ?`,
			seconds(), nodeID, time.Now(), kind, key, nodeID,
		)
	} else {
		query = l.Client.Session.Query(
			`INSERT INTO leases (key, node_id, type, updated_at) VALUES (?, ?, ?, ?)
			        IF NOT EXISTS USING TTL ?`,
			key, nodeID, kind, time.Now(), seconds(),
		)
	}

	query = query.SerialConsistency(gocql.Serial).WithContext(ctx)
	if applied, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		level.Error(l.Logger).Log(
			"message", "error during lease", "error", err, "nodeID", nodeID, "key", key,
		)
		return false
	} else if !applied {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(l.Renew):
			goto start
		}
	}
	return true
}

func (l *Lease) release(key, nodeID string) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), l.DeleteTimeout)
	defer ctxCancel()

	query := l.Client.Session.Query("DELETE FROM leases WHERE key = ? IF node_id = ?", key, nodeID)
	query = query.SerialConsistency(gocql.Serial).WithContext(ctx)

	if _, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		level.Error(l.Logger).Log(
			"message", "error during release lease", "key", key, "nodeID", nodeID, "error", err,
		)
	}
}

func (l *Lease) keepalive(
	ctx context.Context, ctxCancel func(), kind, key, nodeID string, exists bool,
) {
	for {
		select {
		case <-ctx.Done():
			l.release(key, nodeID)
			return
		case <-time.After(l.Renew):
			if l.exec(ctx, kind, key, nodeID, true) {
				continue
			}

			l.release(key, nodeID)
			ctxCancel()
			return
		}
	}
}

func (l *Lease) delay(ctx context.Context, fn func()) {
	if l.nextRun != nil && l.nextRun.Before(*l.nextRun) {
		select {
		case <-time.After(time.Now().Sub(*l.nextRun)):
		case <-ctx.Done():
		}
	}

	defer func() {
		t := time.Now().Add(l.Renew)
		l.nextRun = &t
	}()
	fn()
}

func (l *Lease) nodes(ctx context.Context) ([]string, error) {
	iter := l.Client.Session.Query("SELECT node_id FROM leases WHERE type = ?", leaseRegistry).Iter()

	var (
		id     string
		result []string
	)
	for iter.Scan(&id) {
		result = append(result, id)
	}

	if err := iter.Close(); err != nil {
		return nil, errors.Wrap(err, "error during fetch nodes")
	}
	return result, nil
}
