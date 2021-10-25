package app

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/logger"
	"github.com/euiko/tooyoul/mineman/lib/runner"
)

type App struct {
	config config.Config
}

var registry ModuleRegistry

func (a *App) Run() error {
	// load config
	a.config = config.NewConfigViper("")

	// initialize logger
	l := logger.NewLogrusLogger()
	ctx := logger.InjectContext(context.Background(), l)

	runner.Run(ctx, runner.OperationFunc(func(ctx context.Context) error {
		return a.run(ctx)
	}))

	return nil
}

func (a *App) run(ctx context.Context) error {

	moduleFactories := registry.Load()
	modules := make([]api.Module, len(moduleFactories))

	// instantiate all modules
	for i, f := range moduleFactories {
		modules[i] = f()
	}

	// calls modules init
	for _, m := range modules {
		if err := m.Init(ctx, a.config); err != nil {
			return err
		}
		defer func() {
			if err := m.Close(ctx); err != nil {
				logger.Error(ctx, logger.Message("error while closing module err=", err))
			}
		}()
	}

	return nil
}
