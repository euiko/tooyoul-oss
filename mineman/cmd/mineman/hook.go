package main

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/config"
)

type hook struct {
	ctx    context.Context
	cancel func()
}

func (h *hook) Init(ctx context.Context, c config.Config) error {
	h.ctx, h.cancel = context.WithCancel(ctx)
	return nil
}

func (h *hook) Close(ctx context.Context) error {
	h.cancel()
	return nil
}

func (h *hook) Run(ctx context.Context) app.Waiter {
	errChan := make(chan error)
	go func() {
		<-h.ctx.Done()
		errChan <- nil
	}()
	return app.NewChanWaiter(errChan)
}

func newHook() *hook {
	return &hook{}
}
