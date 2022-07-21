package app

import (
	"context"
	"os"
	"path"
	"syscall"

	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/runner"
)

type App struct {
	config config.Config
	name   string
	hook   Hook

	injectedVals []interface{}
	modules      []api.Module
}

var registry ModuleRegistry

func (a *App) Inject(vals ...interface{}) {
	a.injectedVals = append(a.injectedVals, vals...)
}

func (a *App) Run() error {
	// initialize logger
	l := log.NewLogrusLogger()
	ctx := log.InjectContext(context.Background(), l)
	ctx, cancel := context.WithCancel(ctx)

	// load config
	viperOpts := []config.ViperOptions{}

	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		viperOpts = append(viperOpts,
			config.ViperPaths(homeDir),
			config.ViperPaths(path.Join(homeDir, ".config", a.name)),
		)
	}
	a.config = config.NewViper(a.name, viperOpts...)

	// load logger options
	l.Init(ctx, a.config)
	defer l.Close(ctx)
	log.SetDefault(l)

	runner.Run(ctx, runner.OperationFunc(func(ctx context.Context) error {
		log.Trace("running application...")
		err := a.run(ctx)
		if err != nil {
			log.Error("running app error", log.WithError(err))
		}

		cancel()
		return err
	})).OnSignal(runner.SignalHandlerFunc(func(ctx context.Context, sig os.Signal) {
		if sig == syscall.SIGHUP {
			return
		}

		newCtx := context.TODO()
		for _, m := range a.modules {
			if err := m.Close(newCtx); err != nil {
				log.Error("error when closing modules", log.WithError(err))
				return
			}
		}

		if err := a.hook.Close(newCtx); err != nil {
			log.Error("error when closing hook", log.WithError(err))
			return
		}

		cancel()
		log.Trace("application closed")
	})).Wait(ctx)

	return nil
}

func (a *App) run(ctx context.Context) error {

	moduleFactories := registry.LoadMap()
	modules := []api.Module{}

	log.Trace("initalizing hook...")
	if err := a.hook.Init(ctx, a.config); err != nil {
		return err
	}
	defer a.hook.Close(ctx)
	log.Trace("hook initialized")

	// instantiate all modules
	log.Trace("loading modules...")
	for n, f := range moduleFactories {
		enabled := true
		m := f()

		if h, ok := a.hook.(HookModuleInterceptor); ok {
			enabled = h.Intercept(n, m)
		}

		if !enabled {
			continue
		}

		// call module loaded hook
		if ext, ok := a.hook.(HookModuleExt); ok {
			ext.ModuleLoaded(ctx, m)
		}

		modules = append(modules, m)
	}
	a.modules = modules
	log.Trace("%d modules loaded", log.WithValues(len(modules)))

	// calls modules init
	log.Trace("initializing modules...")
	for _, m := range a.modules {

		if err := m.Init(ctx, a.config); err != nil {
			return err
		}
		if ext, ok := a.hook.(HookModuleExt); ok {
			ext.ModuleInitialized(ctx, m)
		}

		// inject values after module initialized

		defer func(m api.Module) {
			if err := m.Close(ctx); err != nil {
				log.Error("error while closing module", log.WithError(err))
			}
		}(m)
	}
	log.Trace("modules initialized")

	log.Trace("running hook")
	defer log.Trace("hook run done")
	waiter := a.hook.Run(ctx)
	if waiter == nil {
		return nil
	}

	err := <-waiter.Wait()
	return err
}

func New(name string, hooks ...Hook) *App {
	return &App{
		name: name,
		hook: &chainedHook{hooks: hooks},
	}
}

func RegisterModule(name string, factory ModuleFactory) {
	registry.Register(name, factory)
}
