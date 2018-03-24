package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)

type Client struct {
	Key    string
	Secret string
	Region string

	Base *session.Session
}

func (c *Client) Init() error {
	if c.Key == "" {
		return errors.New("missing Key")
	}

	if c.Secret == "" {
		return errors.New("missing Secret")
	}

	if c.Region == "" {
		return errors.New("missing Region")
	}

	return nil
}

func (c *Client) Start() error {
	var err error
	c.Base, err = session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(c.Key, c.Secret, ""),
		Region:      aws.String(c.Region),
	})
	if err != nil {
		return errors.Wrap(err, "error during Session initialization")
	}
	return nil
}
