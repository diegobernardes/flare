package cassandra

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"
)

type Client struct {
	Hosts         []string
	Port          int
	Timeout       time.Duration
	Keyspace      string
	AvoidKeyspace bool

	Session *gocql.Session
}

func (c *Client) Init() error {
	cluster := gocql.NewCluster(c.Hosts...)
	cluster.Port = c.Port
	cluster.Timeout = c.Timeout

	if !c.AvoidKeyspace {
		cluster.Keyspace = c.Keyspace
	}

	var err error
	c.Session, err = cluster.CreateSession()
	if err != nil {
		panic(err)
	}

	return nil
}

// Setup the database.
func (c *Client) Setup() error {
	statements := []string{
		`CREATE KEYSPACE IF NOT EXISTS %s
	   WITH REPLICATION = {
	     'class' : 'SimpleStrategy',
	     'replication_factor' : 1
		 }`,

		`CREATE TABLE IF NOT EXISTS %s.locks (
      key     varchar PRIMARY KEY,
			node_id UUID
		)`,

		`CREATE TABLE IF NOT EXISTS %s.nodes (
			id         UUID PRIMARY KEY,
			created_at timestamp
		)`,

		`CREATE INDEX IF NOT EXISTS created_at ON %s.nodes (created_at)`,

		`CREATE TABLE IF NOT EXISTS %s.consumers (
			id          UUID,
			hash        varchar PRIMARY KEY,
			source_type varchar,
			source      varchar,
			payload     varchar,
			node_id     UUID,
			created_at  timestamp
		)`,

		`CREATE INDEX IF NOT EXISTS consumer_id ON %s.consumers (id)`,

		`CREATE INDEX IF NOT EXISTS created_at ON %s.consumers (created_at)`,

		`CREATE INDEX IF NOT EXISTS consumer_node_id ON %s.consumers (node_id)`,
	}

	for _, statement := range statements {
		statement = fmt.Sprintf(statement, c.Keyspace)
		if err := c.Session.Query(statement).Exec(); err != nil {
			return errors.Wrap(err, "error during setup cassandra")
		}
	}
	return nil
}
