// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// HookResource is used to register and trigger hooks to handle resource states.
type HookResource struct {
	deleteCallbacks []func(context.Context, string) error
}

// Delete is used to trigger all callbacks when a resource is deleted.
func (hr *HookResource) Delete(ctx context.Context, id string) error {
	g, gctx := errgroup.WithContext(ctx)

	for _, hook := range hr.deleteCallbacks {
		g.Go(func(h func(context.Context, string) error) func() error {
			return func() error {
				return h(gctx, id)
			}
		}(hook))
	}

	return g.Wait()
}

// RegisterDelete is used to register the callback to be executed during a resource delete.
func (hr *HookResource) RegisterDelete(callback func(context.Context, string) error) {
	hr.deleteCallbacks = append(hr.deleteCallbacks, callback)
}
