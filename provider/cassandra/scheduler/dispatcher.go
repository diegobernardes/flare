package scheduler

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
	baseConsumer "github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/provider/cassandra"
)

// Dispatcher implements the storage logic to dispatch consumers to be processed.
type Dispatcher struct {
	Client *cassandra.Client
}

// Init check if the struct has everything needed to run.
func (d *Dispatcher) Init() error {
	if d.Client == nil {
		return errors.New("missing Client")
	}
	return nil
}

// Fetch return a list of consumers changed after a period of time.
func (d *Dispatcher) Fetch(ctx context.Context, time *time.Time) ([]consumer.Consumer, error) {
	var query *gocql.Query

	if time == nil {
		query = d.Client.Session.Query(`SELECT id, created_at FROM consumers`)
	} else {
		query = d.Client.Session.Query(
			`SELECT id, created_at FROM consumers WHERE created_at > ? ALLOW FILTERING`, time,
		)
	}

	var (
		iter      = query.WithContext(ctx).Iter()
		consumers []baseConsumer.Consumer
	)

	for {
		var consumer baseConsumer.Consumer
		if ok := iter.Scan(&consumer.ID, &consumer.CreatedAt); !ok {
			break
		}
		consumers = append(consumers, consumer)
	}

	if err := iter.Close(); err != nil {
		return nil, errors.Wrap(err, "error during cassandra iter close")
	}

	return consumers, nil
}

// Assign a given consumer to a node to be processed.
func (d *Dispatcher) Assign(ctx context.Context, consumerID, nodeID string) error {
	// tem que fazer um select antes para pegar o hash
	query := d.Client.Session.Query(
		"UPDATE consumers SET node_id = ? WHERE id = ?", nodeID, consumerID,
	).WithContext(ctx)

	if err := query.Exec(); err != nil {
		return errors.Wrapf(err, "error during assign consumer '%s' to node '%s'", consumerID, nodeID)
	}
	return nil
}

// Unassign the consumer of a node.
func (d *Dispatcher) Unassign(ctx context.Context, consumerID string) error {
	query := d.Client.Session.Query("UPDATE consumers SET node_id = ? WHERE id = ?", nil, consumerID)
	query = query.WithContext(ctx)

	if err := query.Exec(); err != nil {
		return errors.Wrapf(err, "error during unassign consumer '%s'", consumerID)
	}
	return nil
}
