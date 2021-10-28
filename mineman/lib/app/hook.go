package app

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
)

type Waiter interface {
	Wait() <-chan error
}

type Hook interface {
	api.Module
	Run(ctx context.Context) Waiter
}

type HookModuleExt interface {
	ModuleLoaded(ctx context.Context, m api.Module)
	ModuleInitialized(ctx context.Context, m api.Module)
}

type chainedHook struct {
	hooks []Hook
}

type chainedWaiter struct {
	waiters []Waiter
}

type ChanWaiter struct {
	channel chan error
}

func (h *chainedHook) Init(ctx context.Context, c config.Config) error {
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

func (h *chainedHook) Run(ctx context.Context) Waiter {

	w := &chainedWaiter{
		waiters: make([]Waiter, len(h.hooks)),
	}
	for i, h := range h.hooks {
		w.waiters[i] = h.Run(ctx)
	}
	return w
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

func (w *ChanWaiter) Wait() <-chan error {
	return w.channel
}

func NewChanWaiter(channel chan error) *ChanWaiter {
	return &ChanWaiter{
		channel: channel,
	}
}
