package consumer

import (
	"context"
	"time"

	"github.com/diegobernardes/flare/domain/consumer"
)

type fetcher interface {
	Fetch(ctx context.Context, fn func([]consumer.ConsumerSourceAWSSQS) error) error
}

type Client struct {
	Processor func([]byte) error
	Fetcher   fetcher
}

func (c *Client) Start() error {
	if err := c.Fetcher.Fetch(context.Background(), c.scheduler); err != nil {
		panic(err)
	}
	return nil
}

func (c *Client) Stop() error {
	return nil
}

func (c *Client) Init() error {
	return nil
}

func (c *Client) scheduler(sources []consumer.ConsumerSourceAWSSQS) error {
	for _, source := range sources {
		for i := 0; i < source.Concurrency; i++ {
			go c.run(source)
		}
	}
	return nil
}

func (c *Client) run(source consumer.ConsumerSourceAWSSQS) {
	defer func() {
		recover()
		c.run(source)
	}()

	for {
		c.Processor([]byte("maybe its working...."))

		<-time.After(1 * time.Second)
		// tira da fila e chama o c.Processor
	}
}

/*
	vamos chamar essa funcao, se nao der erro, deletamos as mensagens da fila.
	se der erro, tentamos novamente.
*/
