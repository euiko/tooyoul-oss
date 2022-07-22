package app

import (
	"context"
	"fmt"

	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
)

type Waiter interface {
	Wait() <-chan error
}

type Hook interface {
	api.Module
	Run(ctx context.Context) error
}

type HookModuleExt interface {
	ModuleLoaded(ctx context.Context, m api.Module)
	ModuleInitialized(ctx context.Context, m api.Module)
}

// HookModuleInterceptor intercept loading of an module
// you can use this to selectively load/unload module based on hook
// e.g. selectively load modules by platform
type HookModuleInterceptor interface {
	Intercept(name string, module api.Module) bool
}

type chainedHook struct {
	config config.Config
	hooks  []Hook
}

type chainedWaiter struct {
	waiters []Waiter
}

type chanWaiter struct {
	manage  bool
	channel chan error
}

func (h *chainedHook) Init(ctx context.Context, c config.Config) error {
	h.config = c

	for _, h := range h.hooks {
		if err := h.Init(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

func (h *chainedHook) Close(ctx context.Context) error {
	for _, h := range h.hooks {
		if err := h.Close(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (h *chainedHook) Run(ctx context.Context) error {

	waiters := make([]Waiter, len(h.hooks))
	for i, h := range h.hooks {
		errChan := make(chan error, 1)
		w := NewChanWaiter(errChan)

		// run in go routne
		go func(w *chanWaiter, h Hook) {
			w.channel <- h.Run(ctx)
		}(w, h)

		waiters[i] = w
	}

	w := &chainedWaiter{
		waiters: waiters,
	}
	err := <-w.Wait()
	return err
}

func (h *chainedHook) ModuleLoaded(ctx context.Context, m api.Module) {
	for _, h := range h.hooks {
		if ext, ok := h.(HookModuleExt); ok {
			ext.ModuleLoaded(ctx, m)
		}
	}
}

func (h *chainedHook) ModuleInitialized(ctx context.Context, m api.Module) {
	for _, h := range h.hooks {
		if ext, ok := h.(HookModuleExt); ok {
			ext.ModuleInitialized(ctx, m)
		}
	}
}

func (h *chainedHook) Intercept(name string, m api.Module) bool {
	// only one effective interceptor
	var effectiveInterceptor HookModuleInterceptor
	for _, h := range h.hooks {
		if h, ok := h.(HookModuleInterceptor); ok {
			// the first one is effective interceptor
			effectiveInterceptor = h
			break
		}
	}

	// no one, do default
	if effectiveInterceptor == nil {
		return h.defaultInterceptor(name, m)
	}

	return effectiveInterceptor.Intercept(name, m)
}

// defaultInterceptor act as a fallback when there is no interceptor defined in hook
// it will load the module acording its configuration and default loaded module properties
func (h *chainedHook) defaultInterceptor(name string, module api.Module) bool {
	// default loaded
	enabled := true

	// override when it say so
	if dm, ok := module.(api.DefaultModule); ok {
		enabled = dm.Default()
	}

	configKey := fmt.Sprintf("%s.enabled", name)
	return h.config.Get(configKey).Bool(enabled)
}

func (w *chainedWaiter) Wait() <-chan error {
	signalCount := len(w.waiters) - 1
	signalChan := make(chan int, signalCount)
	errChan := make(chan error)
	for i := signalCount; i > 0; i-- {
		signalChan <- 1
	}

	signalOne := func() {
		select {
		case <-signalChan:
			// do nothing
		default:
			// no more signal to be watched, close all the channel
			close(errChan)
			close(signalChan)
		}
	}

	// listen to all closer's Done
	for _, closer := range w.waiters {
		if closer == nil {
			signalOne()
			continue
		}
		go func(closer Waiter) {
			if err := <-closer.Wait(); err != nil {
				// when done close the signal chan
				close(signalChan)
				errChan <- err
			}
			signalOne()
		}(closer)
	}
	return errChan
}

func (w *chanWaiter) Wait() <-chan error {
	if w.manage {
		close(w.channel)
	}
	return w.channel
}

func NewChanWaiter(channel chan error) *chanWaiter {
	return &chanWaiter{
		channel: channel,
		manage:  false,
	}
}

func NewDirectWaiter(initial error) *chanWaiter {
	// use buffered chan, so it can be used in the same thread
	errChan := make(chan error, 1)
	errChan <- initial
	return &chanWaiter{
		channel: errChan,
		manage:  true,
	}
}
