// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import "context"

type pushMock struct {
	err error
}

func (pm *pushMock) push(ctx context.Context, id, action string, body []byte) error {
	if pm.err != nil {
		return pm.err
	}
	return nil
}

func newPushMock(err error) *pushMock {
	return &pushMock{err}
}
