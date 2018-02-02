// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import "github.com/pkg/errors"

// Pagination used to fetch a slice of a given entity.
type Pagination struct {
	Limit  int
	Offset int
	Total  int
}

// Valid indicates if the current pagination is valid.
func (p *Pagination) Valid() error {
	if p.Offset < 0 {
		return errors.Errorf("invalid offset '%d'", p.Offset)
	}

	if p.Limit < 0 {
		return errors.Errorf("invalid limit '%d'", p.Limit)
	}
	return nil
}
