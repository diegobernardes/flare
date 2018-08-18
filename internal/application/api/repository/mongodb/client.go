package mongodb

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/pkg/errors"
)

type Client struct {
	ConnectionString string

	baseClient mongo.Client
}

func (c *Client) Init() error {
	client, err := mongo.NewClient(c.ConnectionString)
	if err != nil {
		return errors.Wrap(err, "error during base client initialization")
	}
	c.baseClient = *client

	err = client.Connect(context.Background())
	return errors.Wrap(err, "error during connection")
}

func (c Client) Database(name string) mongo.Database {
	return *c.baseClient.Database(name)
}
