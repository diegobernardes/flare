// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type config struct {
	content string
	viper   *viper.Viper
}

func (c *config) getString(key string) string { return c.viper.GetString(key) }

func (c *config) getInt(key string) int { return c.viper.GetInt(key) }

func newConfig(options ...func(*config)) (*config, error) {
	c := &config{viper: viper.New()}
	c.viper.SetConfigType("toml")

	for _, option := range options {
		option(c)
	}

	if err := c.viper.ReadConfig(bytes.NewBufferString(c.content)); err != nil {
		return nil, errors.Wrap(err, "error during config setup")
	}

	return c, nil
}

func configContent(content string) func(*config) {
	return func(c *config) { c.content = content }
}
