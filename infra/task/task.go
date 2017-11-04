// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

import "context"

// Pusher is used to send a task to be processed.
type Pusher interface {
	Push(context.Context, []byte) error
}

// Puller is used to fetch a task to process.
type Puller interface {
	Pull(context.Context, func(context.Context, []byte) error) error
}

// Processer is used to process the tasks.
type Processer interface {
	Process(context.Context, []byte) error
}
