// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"

	"github.com/diegobernardes/flare"
)

type pushMock struct {
	err error
}

func (pm *pushMock) push(ctx context.Context, action string, doc *flare.Document) error {
	if pm.err != nil {
		return pm.err
	}
	return nil
}

func newPushMock(err error) *pushMock {
	return &pushMock{err}
}

type pushWorkerMock struct {
	err error
}

func (pm *pushWorkerMock) Push(context.Context, []byte) error {
	if pm.err != nil {
		return pm.err
	}
	return nil
}

func newPushWorkerMock(err error) *pushWorkerMock {
	return &pushWorkerMock{err}
}
