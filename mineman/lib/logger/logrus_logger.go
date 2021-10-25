package logger

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/sirupsen/logrus"
)

type LogrusLogger struct {
	logrus *logrus.Logger
	option Option
}

func (l *LogrusLogger) toLogrusLevel(level Level) logrus.Level {
	return logrus.Level(l.option.Level + 1)
}

func (l *LogrusLogger) Init(ctx context.Context, c config.Config) error {
	l.option = LoadOption(c)
	l.logrus = logrus.New()
	l.logrus.SetLevel(l.toLogrusLevel(Level(l.option.Level)))
	return nil
}

func (l *LogrusLogger) Close(ctx context.Context) error {
	return nil
}

func (l *LogrusLogger) Log(level Level, msg *MessageLog) {
	l.logrus.Log(l.toLogrusLevel(level), msg.message)
}

func NewLogrusLogger() *LogrusLogger {
	return &LogrusLogger{logrus: logrus.New()}
}
