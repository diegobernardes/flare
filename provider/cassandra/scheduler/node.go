package scheduler

import (
	"context"
	"time"

	"github.com/gocql/gocql"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/provider/cassandra"
	"github.com/diegobernardes/flare/scheduler"
)

type Node struct {
	Client *cassandra.Client
}

// Join is used to set the current node into a list of nodes of the cluster.
func (n *Node) Join(ctx context.Context, id string, ttl time.Duration) error {
	query := n.Client.Session.Query(
		`INSERT INTO nodes (id, created_at) VALUES (?, ?) IF NOT EXISTS USING TTL ?`,
		id, time.Now(), (int64)(ttl.Seconds()),
	).WithContext(ctx)

	if applied, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "error during insert node")
	} else if !applied {
		return errors.New("could not insert node")
	}
	return nil
}

func (n *Node) KeepAlive(ctx context.Context, id string, ttl time.Duration) error {
	query := n.Client.Session.Query(`
		UPDATE nodes USING TTL ? SET created_at = ? WHERE id = ?`,
		(int64)(ttl.Seconds()), time.Now(), id,
	).WithContext(ctx)

	if err := query.Exec(); err != nil {
		return errors.Wrap(err, "error during node keep alive")
	}
	return nil
}

func (n *Node) Leave(ctx context.Context, id string) error {
	query := n.Client.Session.Query(`DELETE from nodes WHERE id = ?`, id).WithContext(ctx)

	if err := query.Exec(); err != nil {
		return errors.Wrap(err, "error during node delete")
	}
	return nil
}

func (n *Node) Nodes(ctx context.Context, time *time.Time) ([]scheduler.Node, error) {
	var query *gocql.Query

	if time == nil {
		query = n.Client.Session.Query("SELECT id, created_at FROM nodes")
	} else {
		query = n.Client.Session.Query("SELECT id, created_at FROM nodes WHERE created_at >= ?", time)
	}

	var (
		iter    = query.WithContext(ctx).Iter()
		results []scheduler.Node
	)

	for {
		var result scheduler.Node
		if ok := iter.Scan(&result.ID, &result.CreatedAt); !ok {
			break
		}
		results = append(results, result)
	}

	if err := iter.Close(); err != nil {
		return nil, errors.Wrap(err, "error during fetch nodes")
	}
	return results, nil
}

// Init check if the struct has everything needed to run.
func (n *Node) Init() error {
	if n.Client == nil {
		return errors.New("missing Client")
	}
	return nil
}
