// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
	"github.com/diegobernardes/flare/subscription"
)

// Trigger implements flare.SubscriptionTrigger.
type Trigger struct {
	base *subscription.Trigger
	err  error
}

// Update mock flare.DocumentRepositorier.Update.
func (t *Trigger) Update(ctx context.Context, document *flare.Document) error {
	if t.err != nil {
		return t.err
	}
	return t.base.Update(ctx, document)
}

// Delete mock flare.DocumentRepositorier.Delete.
func (t *Trigger) Delete(ctx context.Context, document *flare.Document) error {
	if t.err != nil {
		return t.err
	}
	return t.base.Delete(ctx, document)
}

// NewTrigger return a flare.SubscriptionTrigger mock.
func NewTrigger(options ...func(*Trigger)) (*Trigger, error) {
	trigger, err := subscription.NewTrigger(
		subscription.TriggerHTTPClient(http.DefaultClient),
		subscription.TriggerRepository(memory.NewSubscription()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "errors during Trigger initialization")
	}
	t := &Trigger{base: trigger}

	for _, option := range options {
		option(t)
	}

	return t, nil
}

// TriggerError set the error to be returned during calls.
func TriggerError(err error) func(*Trigger) {
	return func(t *Trigger) { t.err = err }
}
