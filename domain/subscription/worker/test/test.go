// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"context"

	"github.com/diegobernardes/flare"
)

// Trigger is a mock of the subscription.Trigger, this is used by tests.
type Trigger struct{ err error }

// Push is a mock of subscription.Push, this is used by tests.
func (t *Trigger) Push(_ context.Context, _ *flare.Document, _ string) error { return t.err }

// NewTrigger returns a configured mock trigger.
func NewTrigger(err error) *Trigger {
	return &Trigger{err}
}
