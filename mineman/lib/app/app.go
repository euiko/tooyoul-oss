package app

import (
	"context"
	"os"
	"syscall"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/logger"
	"github.com/euiko/tooyoul/mineman/lib/runner"
)

type App struct {
	config config.Config
	name   string
	hook   Hook
}

var registry ModuleRegistry

func RegisterModule(name string, factory ModuleFactory) {
	registry.Register(name, factory)
}

func (a *App) Run() error {
	// initialize logger
	l := logger.NewLogrusLogger()
	ctx := logger.InjectContext(context.Background(), l)
	ctx, cancel := context.WithCancel(ctx)

	logger.Trace(ctx, logger.Message("loading config..."))
	// load config
	a.config = config.NewViper(a.name)
	logger.Trace(ctx, logger.Message("config loaded"))

	// load logger options
	// l.Init(ctx, a.config)
	// defer l.Close(ctx)

	runner.Run(ctx, runner.OperationFunc(func(ctx context.Context) error {
		logger.Trace(ctx, logger.Message("running application..."))
		err := a.run(ctx)
		if err != nil {
			logger.Error(ctx, logger.Message("running app error with err=", err))
		}

		cancel()
		return err
	})).OnSignal(runner.SignalHandlerFunc(func(ctx context.Context, sig os.Signal) {
		logger.Trace(ctx, logger.Message("application closed"))
		if sig == syscall.SIGHUP {
			return
		}

		if err := a.hook.Close(ctx); err != nil {
			logger.Error(ctx, logger.Message("error when closing the hook"))
		}

	})).Wait(ctx)

	return nil
}

func (a *App) run(ctx context.Context) error {

	moduleFactories := registry.Load()
	modules := make([]api.Module, len(moduleFactories))

	logger.Trace(ctx, logger.Message("initalizing hook..."))
	if err := a.hook.Init(ctx, a.config); err != nil {
		return err
	}
	defer a.hook.Close(ctx)
	logger.Trace(ctx, logger.Message("hook initialized"))

	// instantiate all modules
	logger.Trace(ctx, logger.Message("loading modules..."))
	for i, f := range moduleFactories {
		m := f()
		if ext, ok := a.hook.(HookModuleExt); ok {
			ext.ModuleLoaded(ctx, m)
		}
		modules[i] = m
	}
	logger.Trace(ctx, logger.Message("modules loaded"))

	// calls modules init
	logger.Trace(ctx, logger.Message("initializing modules..."))
	for _, m := range modules {
		if err := m.Init(ctx, a.config); err != nil {
			return err
		}
		if ext, ok := a.hook.(HookModuleExt); ok {
			ext.ModuleInitialized(ctx, m)
		}
		defer func(m api.Module) {
			if err := m.Close(ctx); err != nil {
				logger.Error(ctx, logger.Message("error while closing module err=", err))
			}
		}(m)
	}
	logger.Trace(ctx, logger.Message("modules initialized"))

	logger.Trace(ctx, logger.Message("running hook"))
	defer logger.Trace(ctx, logger.Message("hook run done"))
	waiter := a.hook.Run(ctx)
	if waiter == nil {
		return nil
	}

	return <-waiter.Wait()
}

func New(name string, hooks ...Hook) *App {
	return &App{
		name: name,
		hook: &chainedHook{hooks: hooks},
	}
}
