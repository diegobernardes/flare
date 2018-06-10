// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/kr/pretty"
)

type queue interface {
	Push(ctx context.Context, payload []byte) error
	Pull(ctx context.Context, fn func(context.Context, []byte) error) error
}

// Client implements the queue interface.
type Client struct {
	mutex         sync.Mutex
	messages      [][]byte
	RegisterQueue func(queue) error
}

func (c *Client) Create(ctx context.Context, id string) error {
	pretty.Println("chamei aqui...")
	os.Exit(1)

	if err := c.RegisterQueue(c); err != nil {
		panic(err)
	}

	// mas e ai, oq eue faco aqui, crio uma fila e registro em algum lugar?
	// onde posso registrar essa merda!?
	// queue register? toda vez que criar uma fila, registrar no queue register.
	// se o queu register registrar, enviar pro queue notifier.
	// dai temos que ter o worker em algum lugar que vai ouvir e entao comecar a processar a fila.

	// o register precisa ter uma interface para a fila, se nao vai dar merda.

	return nil
}

// Push the message to queue.
func (c *Client) Push(_ context.Context, content []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.messages = append(c.messages, content)
	return nil
}

// Pull fetch a message from queue.
func (c *Client) Pull(ctx context.Context, fn func(context.Context, []byte) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.messages) == 0 {
		<-time.After(100 * time.Millisecond)
		return nil
	}

	if err := fn(ctx, c.messages[0]); err != nil {
		return err
	}
	c.messages = c.messages[1:]
	return nil
}

// NewClient return a configured client.
func NewClient(options ...func(*Client)) *Client {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	return c
}
