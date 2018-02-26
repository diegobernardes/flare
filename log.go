package flare

import (
	"fmt"
	"os"

	base "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log/term"

	"github.com/diegobernardes/flare/infra/config"
)

const (
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

type log struct {
	config *config.Client
	base   base.Logger
}

func (l *log) init() error {
	logger, err := l.initOutput()
	if err != nil {
		return err
	}

	logger, err = l.initLevel(logger)
	if err != nil {
		return err
	}

	l.base = base.With(logger, "time", base.DefaultTimestamp)
	return nil
}

func (l *log) initOutput() (base.Logger, error) {
	output := l.config.GetString("log.output")
	switch output {
	case "discard":
		return base.NewNopLogger(), nil
	case "stdout":
		format := l.config.GetString("log.format")
		switch format {
		case "human":
			return term.NewLogger(base.NewSyncWriter(os.Stdout), base.NewLogfmtLogger, l.color), nil
		case "json":
			return base.NewJSONLogger(base.NewSyncWriter(os.Stdout)), nil
		default:
			return nil, fmt.Errorf("invalid log.format config '%s'", format)
		}
	default:
		return nil, fmt.Errorf("invalid log.output config '%s'", output)
	}
}

func (l *log) initLevel(logger base.Logger) (base.Logger, error) {
	logLevel := l.config.GetString("log.level")
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

func (l *log) color(keyvals ...interface{}) term.FgBgColor {
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
