// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import "context"

type Client struct{}

func (c *Client) Init() error {
	return nil
}

func (c *Client) Create(ctx context.Context, id string) error {
	// posso retornar um sqs client aqui.

	return nil
}
