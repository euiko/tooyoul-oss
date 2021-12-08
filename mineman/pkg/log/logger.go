package log

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
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

type key int

var (
	loggerContextKey key
	fieldsContextKey key
	ErrNoLogger      = errors.New("no logger available")
)

type (
	// Logger represent any logging capable
	Logger interface {
		api.Module
		SetLevel(level Level)
		Log(level Level, msg *MessageLog)
	}

	// MessageLog hold message to be logged
	MessageLog struct {
		ts      time.Time
		message string
		fields  map[string]interface{}
		err     error
	}

	// Config hold logger configuration
	Config struct {
		Level int `mapstructure:"level"`
	}

	Option struct {
		logger       Logger
		ctx          context.Context
		msg          *MessageLog
		formatValues []interface{}
	}

	Options interface {
		Configure(o *Option)
	}

	OptionsFunc func(o *Option)
)

var globalLogger Logger

func (f OptionsFunc) Configure(o *Option) {
	f(o)
}

// WithFileds adds all parameter fields for decorate current message option
func WithFields(fields map[string]interface{}) Options {
	return OptionsFunc(func(o *Option) {
		for k, v := range fields {
			o.msg.fields[k] = v
		}
	})
}

// WithField add a single field to the message option
func WithField(key string, value interface{}) Options {
	return OptionsFunc(func(o *Option) {
		o.msg.fields[key] = value
	})
}

// WithContext decorate current message option to include a context
// maybe override the logger option
func WithContext(ctx context.Context) Options {
	return OptionsFunc(func(o *Option) {
		o.ctx = ctx
	})
}

// WithLogger decorate message option to use specific logger
func WithLogger(logger Logger) Options {
	return OptionsFunc(func(o *Option) {
		o.logger = logger
	})
}

// WithValues decorate message option to use Printf style formatting
func WithValues(v ...interface{}) Options {
	return OptionsFunc(func(o *Option) {
		o.formatValues = v[:]
	})
}

// WithTime decorate message option that override default now timestamp
func WithTime(t time.Time) Options {
	return OptionsFunc(func(o *Option) {
		o.msg.ts = t
	})
}

// WithTime decorate message option that override the error
func WithError(err error) Options {
	return OptionsFunc(func(o *Option) {
		o.msg.err = err
	})
}

func Fatal(msg string, opts ...Options) error {
	return log(FatalLevel, msg, opts...)
}

func Error(msg string, opts ...Options) error {
	return log(ErrorLevel, msg, opts...)
}

func Warning(msg string, opts ...Options) error {
	return log(WarningLevel, msg, opts...)
}

func Info(msg string, opts ...Options) error {
	return log(InfoLevel, msg, opts...)
}

func Debug(msg string, opts ...Options) error {
	return log(DebugLevel, msg, opts...)
}

func Trace(msg string, opts ...Options) error {
	return log(TraceLevel, msg, opts...)
}

func newMessageOption(message string, options ...Options) Option {

	// instantiate message
	msg := &MessageLog{
		ts:      time.Now(),
		message: message,
		fields:  make(map[string]interface{}),
	}

	// set default loggin option
	opt := Option{
		msg:          msg,
		logger:       globalLogger,
		ctx:          nil,
		formatValues: nil,
	}

	// load all decorator function
	for _, o := range options {
		o.Configure(&opt)
	}

	return opt
}

func log(level Level, msg string, opts ...Options) error {
	// global logger not yet specified, then do nothing
	if globalLogger == nil {
		return nil
	}

	opt := newMessageOption(msg, opts...)
	logger := opt.logger

	// load all context specific option
	if opt.ctx != nil {
		ctx := opt.ctx

		// override logger when exists in context
		l := FromContext(ctx)
		if l != nil {
			logger = l
		}

		// add additional fields defined in context
		// override all previously defined option
		fields := FieldsFromContext(ctx)
		for k, v := range fields {
			opt.msg.fields[k] = v
		}
	}

	// format log message when values exists
	if len(opt.formatValues) > 0 {
		opt.msg.message = fmt.Sprintf(opt.msg.message, opt.formatValues...)
	}

	// do log
	logger.Log(level, opt.msg)
	return nil
}

// LoadConfig help you to load logger's config from the general config
func LoadConfig(c config.Config) Config {
	var conf Config
	c.Get("logger").Scan(&conf)
	return conf
}

func SetLevel(level Level) {
	logger := Default()
	logger.SetLevel(level)
}

func SetDefault(logger Logger) {
	globalLogger = logger
}

func Default() Logger {
	return globalLogger
}

func InjectFieldsContext(ctx context.Context, fields map[string]interface{}) context.Context {
	// load or create fields from context
	current := FieldsFromContext(ctx)
	if current == nil {
		current = make(map[string]interface{})
	}

	// merge all fields
	for k, v := range fields {
		current[k] = v
	}

	// inject to context
	return context.WithValue(ctx, loggerContextKey, current)
}

func FieldsFromContext(ctx context.Context) map[string]interface{} {
	instance := ctx.Value(fieldsContextKey)
	if instance == nil {
		return nil
	}

	l, ok := instance.(map[string]interface{})
	if !ok {
		return nil
	}

	return l
}

func InjectContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

func FromContext(ctx context.Context) Logger {
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
