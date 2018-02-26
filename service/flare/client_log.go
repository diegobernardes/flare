// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log/term"
)

const (
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

func (c *Client) initLogger() error {
	logger, err := c.initLoggerOutput()
	if err != nil {
		return err
	}

	logger, err = c.initLoggerLevel(logger)
	if err != nil {
		return err
	}

	c.logger = log.With(logger, "time", log.DefaultTimestamp)
	c.loggerInfo = level.Info(c.logger)
	return nil
}

func (c *Client) initLoggerOutput() (log.Logger, error) {
	output := c.config.GetString("log.output")
	switch output {
	case "discard":
		return log.NewNopLogger(), nil
	case "stdout":
		format := c.config.GetString("log.format")
		switch format {
		case "human":
			return term.NewLogger(log.NewSyncWriter(os.Stdout), log.NewLogfmtLogger, c.loggerColor), nil
		case "json":
			return log.NewJSONLogger(log.NewSyncWriter(os.Stdout)), nil
		default:
			return nil, fmt.Errorf("invalid log.format config '%s'", format)
		}
	default:
		return nil, fmt.Errorf("invalid log.output config '%s'", output)
	}
}

func (c *Client) initLoggerLevel(logger log.Logger) (log.Logger, error) {
	logLevel := c.config.GetString("log.level")
	var filter level.Option

	switch logLevel {
	case logLevelDebug:
		filter = level.AllowDebug()
	case logLevelInfo:
		filter = level.AllowInfo()
	case logLevelWarn:
		filter = level.AllowWarn()
	case logLevelError:
		filter = level.AllowError()
	default:
		return nil, fmt.Errorf("invalid log.level config '%s'", logLevel)
	}

	return level.NewFilter(logger, filter), nil
}

func (c *Client) loggerColor(keyvals ...interface{}) term.FgBgColor {
	for i := 0; i < len(keyvals)-1; i += 2 {
		if keyvals[i] != "level" {
			continue
		}

		switch keyvals[i+1].(level.Value).String() {
		case logLevelDebug:
			return term.FgBgColor{Fg: term.DarkGray}
		case logLevelInfo:
			return term.FgBgColor{Fg: term.Gray}
		case logLevelWarn:
			return term.FgBgColor{Fg: term.Yellow}
		case logLevelError:
			return term.FgBgColor{Fg: term.Red}
		default:
			return term.FgBgColor{}
		}
	}

	return term.FgBgColor{}
}
