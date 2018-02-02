// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

type errMemory struct {
	message       string
	alreadyExists bool
	notFound      bool
}

func (e *errMemory) Error() string       { return e.message }
func (e *errMemory) AlreadyExists() bool { return e.alreadyExists }
func (e *errMemory) NotFound() bool      { return e.notFound }
