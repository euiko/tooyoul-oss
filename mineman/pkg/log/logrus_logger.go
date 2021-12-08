package log

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/sirupsen/logrus"
)

type LogrusLogger struct {
	logrus *logrus.Logger
	option Config
}

func (l *LogrusLogger) toLogrusLevel(level Level) logrus.Level {
	return logrus.Level(level + 1)
}

func (l *LogrusLogger) Init(ctx context.Context, c config.Config) error {
	l.option = LoadConfig(c)
	l.logrus = logrus.New()
	l.logrus.SetLevel(l.toLogrusLevel(Level(l.option.Level)))
	return nil
}

func (l *LogrusLogger) Close(ctx context.Context) error {
	return nil
}

func (l *LogrusLogger) SetLevel(level Level) {
	l.logrus.SetLevel(l.toLogrusLevel(level))
}

func (l *LogrusLogger) Log(level Level, msg *MessageLog) {
	entry := l.logrus.WithFields(msg.fields).
		WithTime(msg.ts)

	// only add error when is not nil
	if msg.err != nil {
		entry = entry.WithError(msg.err)
	}

	entry.Log(l.toLogrusLevel(level), msg.message)
}

func NewLogrusLogger() *LogrusLogger {
	l := logrus.New()
	l.SetLevel(logrus.TraceLevel)
	return &LogrusLogger{logrus: l}
}

func init() {
	SetDefault(NewLogrusLogger())
}
