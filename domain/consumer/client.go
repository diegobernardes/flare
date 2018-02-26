// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package consumer

import "github.com/kr/pretty"

// Client is used to start all the consumers.
type Client struct{}

// Start is used to start the consumers.
func (c *Client) Start() error {
	// vai chamar o scheduler e ele que vai iniciar a merda toda.
	return nil
}

// TODO: temos que receber o consumer por parametro tambem.
func (c *Client) Process(consumer Consumer, payload []byte) error {
	pretty.Println("message: ", string(payload))
	// all new messages come here!
	return nil
}
