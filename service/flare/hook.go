// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import "github.com/diegobernardes/flare"

type hook struct {
	resource     *flare.HookResource
	subscription *flare.HookSubscription
}

func (h *hook) init() {
	h.resource = &flare.HookResource{}
	h.subscription = &flare.HookSubscription{}
}
