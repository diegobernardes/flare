// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// HookSubscription is used to register and trigger hooks to handle subscription states.
type HookSubscription struct {
	createCallbacks []func(context.Context, *Subscription) error
}

// Create is used to trigger all callbacks when a subscription is created.
func (hs *HookSubscription) Create(ctx context.Context, s *Subscription) error {
	g, gctx := errgroup.WithContext(ctx)

	for _, hook := range hs.createCallbacks {
		g.Go(func(h func(context.Context, *Subscription) error) func() error {
			return func() error {
				return h(gctx, s)
			}
		}(hook))
	}

	return g.Wait()
}

// RegisterCreate is used to register the callback to be executed during a subscription create.
func (hs *HookSubscription) RegisterCreate(callback func(context.Context, *Subscription) error) {
	hs.createCallbacks = append(hs.createCallbacks, callback)
}
