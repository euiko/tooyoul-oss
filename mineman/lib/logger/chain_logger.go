package logger

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/config"
)

type LoggerFactory func() Logger

type ChainLogger struct {
	factories  []LoggerFactory
	initialize bool

	loggers []Logger
}

func (l *ChainLogger) Init(ctx context.Context, c config.Config) error {

	// load all loggers first
	loggers := make([]Logger, len(l.factories))
	for i, f := range l.factories {
		loggers[i] = f()
	}

	// initialize all loggers
	if l.initialize {
		for _, l := range loggers {
			l.Init(ctx, c)
		}
	}

	l.loggers = loggers
	return nil
}

func (l *ChainLogger) Close(ctx context.Context) error {

	// close all loggers when initialize defined
	if l.initialize {
		for _, l := range l.loggers {
			err := l.Close(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *ChainLogger) Log(level Level, msg *MessageLog) {
	// log through all available logger
	for _, l := range l.loggers {
		l.Log(level, msg)
	}
}
