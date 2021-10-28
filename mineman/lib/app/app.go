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

func (a *App) Run() error {
	// initialize logger
	l := logger.NewLogrusLogger()
	ctx := logger.InjectContext(context.Background(), l)
	ctx, cancel := context.WithCancel(ctx)

	logger.Trace(ctx, logger.Message("loading config..."))
	// load config
	a.config = config.NewViper(a.name)
	logger.Trace(ctx, logger.Message("config loaded"))

	runner.Run(ctx, runner.OperationFunc(func(ctx context.Context) error {
		err := a.run(ctx)
		if err != nil {
			logger.Error(ctx, logger.Message("running app error with err=", err))
		}

		cancel()
		return err
	})).OnSignal(runner.SignalHandlerFunc(func(ctx context.Context, sig os.Signal) {
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

func New(name string, hook Hook) *App {
	return &App{
		name: name,
		hook: hook,
	}
}
