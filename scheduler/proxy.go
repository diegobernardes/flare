// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"github.com/pkg/errors"
)

type Proxy struct {
	Runners []Runner
}

func (p *Proxy) Start() {
	for _, runner := range p.Runners {
		runner.Start()
	}
}

func (p *Proxy) Stop() {
	for _, runner := range p.Runners {
		runner.Stop()
	}
}

func (p *Proxy) Init() error {
	if len(p.Runners) == 0 {
		return errors.New("missing Runners")
	}
	return nil
}
