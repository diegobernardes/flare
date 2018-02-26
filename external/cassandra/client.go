package cassandra

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"
)

// Client used to access Cassandra database.
type Client struct {
	Hosts         []string
	Port          int
	Keyspace      string
	AvoidKeyspace bool
	Timeout       time.Duration

	Session *gocql.Session
}

// Init check the parameters to initialize the connection.
func (c *Client) Init() error {
	if len(c.Hosts) == 0 {
		return errors.New("missing Hosts")
	}

	if c.Port < 0 || c.Port > 65535 {
		return errors.New("invalid Port")
	}

	if c.Timeout < 0 {
		return errors.New("invalid Timeout")
	}

	return nil
}

// Start initialize the connection.
func (c *Client) Start() error {
	cluster := gocql.NewCluster(c.Hosts...)
	cluster.Port = c.Port

	if !c.AvoidKeyspace {
		cluster.Keyspace = c.Keyspace
	}

	if c.Timeout > 0 {
		cluster.Timeout = c.Timeout
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}

	c.Session = session
	return nil
}

// Stop the connection.
func (c *Client) Stop() { c.Session.Close() }

// Setup the database.
func (c *Client) Setup() error {
	statements := []string{
		`CREATE KEYSPACE IF NOT EXISTS %s
	   WITH REPLICATION = {
	     'class' : 'SimpleStrategy',
	     'replication_factor' : 1
		 }`,

		`CREATE TABLE IF NOT EXISTS %s.leases (
      key        varchar PRIMARY KEY,
			node_id    UUID,
			type       varchar,
			updated_at timestamp
		)`,

		`CREATE INDEX IF NOT EXISTS type ON %s.leases (type)`,

		`CREATE TABLE IF NOT EXISTS %s.consumers (
			id          UUID,
			hash        varchar PRIMARY KEY,
			source_type varchar,
			source      varchar,
			payload     varchar,
			node_id     UUID,
			created_at  timestamp
		)`,

		`CREATE INDEX IF NOT EXISTS consumers_id ON %s.consumers (id)`,

		`CREATE INDEX IF NOT EXISTS consumers_created_at ON %s.consumers (created_at)`,

		`CREATE INDEX IF NOT EXISTS consumers_node_id ON %s.consumers (node_id)`,
	}

	for _, statement := range statements {
		statement = fmt.Sprintf(statement, c.Keyspace)
		if err := c.Session.Query(statement).Exec(); err != nil {
			return errors.Wrap(err, "error during setup")
		}
	}
	return nil
}
