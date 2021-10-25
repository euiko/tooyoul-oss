package runner

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type (
	SignalNotifier interface {
		Wait(context.Context)
		OnSignal(handler SignalHandler) SignalNotifier
	}

	SignalHandler interface {
		Handle(ctx context.Context, sig os.Signal)
	}

	SignalHandlerFunc func(ctx context.Context, sig os.Signal)

	signalInterceptor struct {
		handlers []SignalHandler
	}
)

func (h SignalHandlerFunc) Handle(ctx context.Context, sig os.Signal) {
	h(ctx, sig)
}

func (i *signalInterceptor) Wait(ctx context.Context) {
	sink := make(chan os.Signal, 1)
	defer close(sink)

	// wait for signal
	signal.Notify(sink, signals...)

	// reset the watched signals
	defer signal.Ignore(signals...)

	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sink:
			if sig != syscall.SIGHUP {
				i.callHandlers(ctx, sig)
				return
			}

			i.callHandlers(ctx, sig)
		}
	}
}

func (i *signalInterceptor) OnSignal(handler SignalHandler) SignalNotifier {
	i.handlers = append(i.handlers, handler)
	return i
}

func (i *signalInterceptor) callHandlers(ctx context.Context, sig os.Signal) {
	for _, h := range i.handlers {
		h.Handle(ctx, sig)
	}
}
