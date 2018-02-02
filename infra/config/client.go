// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Client used to access the configurations.
type Client struct {
	Content string
	viper   *viper.Viper
}

// Set the value of a key.
func (c *Client) Set(key string, value interface{}) { c.viper.Set(key, value) }

// IsSet check if a given configuration exists.
func (c *Client) IsSet(key string) bool { return c.viper.IsSet(key) }

// GetString get the value as a string from a given configuration.
func (c *Client) GetString(key string) string { return c.viper.GetString(key) }

// GetStringSlice get the value as a []string from a given configuration.
func (c *Client) GetStringSlice(key string) []string { return c.viper.GetStringSlice(key) }

// GetInt get the value as a int from a given configuration.
func (c *Client) GetInt(key string) int { return c.viper.GetInt(key) }

// GetBool get the value as a bool from a given configuration.
func (c *Client) GetBool(key string) bool { return c.viper.GetBool(key) }

// GetDuration get the value as a time.Duration from a given configuration.
func (c *Client) GetDuration(key string) (time.Duration, error) {
	value := c.GetString(key)
	if value == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("error during parse '%s' to time.Duration", key))
	}
	return duration, nil
}

// Init the client instance.
func (c *Client) Init() error {
	c.viper = viper.New()
	c.viper.SetConfigType("toml")

	if err := c.viper.ReadConfig(bytes.NewBufferString(c.Content)); err != nil {
		return errors.Wrap(err, "error during config parse")
	}
	return nil
}
