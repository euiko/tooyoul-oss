package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
)

type Level int

const (
	FatalLevel Level = iota
	ErrorLevel
	WarningLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

const (
	loggerContextKey = "logger"
)

var (
	ErrNoLogger = errors.New("no logger available")
)

type (
	Logger interface {
		api.Module
		Log(level Level, msg *MessageLog)
	}

	MessageLog struct {
		ts      time.Time
		message string
		fields  map[string]interface{}
	}

	Option struct {
		Level int `mapstructure:"level"`
	}
)

func LoadOption(c config.Config) Option {
	var conf Option
	c.Get("logger").Scan(&conf)
	return conf
}

func Message(a ...interface{}) *MessageLog {
	return &MessageLog{
		ts:      time.Now(),
		message: fmt.Sprint(a...),
		fields:  make(map[string]interface{}),
	}
}
func MessageFmt(format string, v ...interface{}) *MessageLog {
	return Message(fmt.Sprintf(format, v...))
}

func (l *MessageLog) WithFields(fields map[string]interface{}) *MessageLog {
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

func InjectContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

func Fatal(ctx context.Context, msg *MessageLog) error {
	return log(ctx, FatalLevel, msg)
}

func Error(ctx context.Context, msg *MessageLog) error {
	return log(ctx, ErrorLevel, msg)
}

func Warning(ctx context.Context, msg *MessageLog) error {
	return log(ctx, WarningLevel, msg)
}

func Info(ctx context.Context, msg *MessageLog) error {
	return log(ctx, InfoLevel, msg)
}

func Debug(ctx context.Context, msg *MessageLog) error {
	return log(ctx, DebugLevel, msg)
}

func Trace(ctx context.Context, msg *MessageLog) error {
	return log(ctx, TraceLevel, msg)
}

func log(ctx context.Context, level Level, msg *MessageLog) error {
	l := getLogger(ctx)
	if l == nil {
		return ErrNoLogger
	}

	l.Log(level, msg)
	return nil
}

func getLogger(ctx context.Context) Logger {
	instance := ctx.Value(loggerContextKey)
	if instance == nil {
		return nil
	}

	l, ok := instance.(Logger)
	if !ok {
		return nil
	}

	return l
}
