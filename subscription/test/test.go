// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"context"

	"github.com/diegobernardes/flare"
)

// Trigger is a mock of the subscription.Trigger, this is used by tests.
type Trigger struct {
	err error
}

// Update is the mock of subscription.Trigger.Update.
func (t *Trigger) Update(ctx context.Context, document *flare.Document) error {
	return t.err
}

// Delete is the mock of subscription.Trigger.Delete.
func (t *Trigger) Delete(ctx context.Context, document *flare.Document) error {
	return t.err
}

// NewTrigger returns a configured mock trigger.
func NewTrigger(err error) *Trigger {
	return &Trigger{err}
}
