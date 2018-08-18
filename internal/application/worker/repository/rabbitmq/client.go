package rabbitmq

import "github.com/streadway/amqp"

type Client struct{}

func (c Client) Init() error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(err)
	}
	defer conn.Close() // nolint

	channel, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	_ = channel

	// tem que suportar o cancel para o modo async

	// channel.Consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table)

	// need to do this 2.
	// channel.QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp.Table)
	// channel.ExchangeDeclare(name string, kind string, durable bool, autoDelete bool, internal bool, noWait bool, args amqp.Table)

	return nil
}
