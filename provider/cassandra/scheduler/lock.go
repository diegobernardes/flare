package scheduler

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/provider/cassandra"
)

// Lock implements the repository to be used by scheduler.
type Lock struct {
	Client *cassandra.Client
}

// Init check if the struct has everything needed to run.
func (l *Lock) Init() error {
	if l.Client == nil {
		return errors.New("missing client")
	}
	return nil
}

// Lock is used to lock a given key during a period of time.
func (l *Lock) Lock(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error) {
	return l.acquire(ctx, key, nodeID, ttl, false)
}

// Refresh is used to refresh the lock on a given key for a period of time.
func (l *Lock) Refresh(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error) {
	return l.acquire(ctx, key, nodeID, ttl, true)
}

// Release remove the lock from a node at a given key.
func (l *Lock) Release(ctx context.Context, key, nodeID string) error {
	query := l.Client.Session.Query("DELETE FROM LOCKS WHERE key = ? IF node_id = ?", key, nodeID)
	query = query.SerialConsistency(gocql.Serial)

	if _, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		return errors.Wrapf(err, "error during release lock with key '%s' from node '%s'", key, nodeID)
	}
	return nil
}

func (l *Lock) acquire(
	ctx context.Context, key, nodeID string, ttl time.Duration, exists bool,
) (bool, error) {
	var query *gocql.Query

	if exists {
		query = l.Client.Session.Query(
			"UPDATE locks USING TTL ? SET node_id = ? WHERE key = ? IF node_id = ?",
			(int)(ttl.Seconds()), nodeID, key, nodeID,
		)
	} else {
		query = l.Client.Session.Query(
			"INSERT INTO locks (key, node_id) VALUES (?, ?) IF NOT EXISTS USING TTL ?",
			key, nodeID, (int)(ttl.Seconds()),
		)
	}

	query = query.SerialConsistency(gocql.Serial)
	if applied, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		return false, errors.Wrap(err, "error during lock")
	} else if !applied {
		return false, nil
	}
	return true, nil
}
