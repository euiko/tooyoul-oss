package event

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/app"
	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/log"
)

type (
	HookConfig struct {
		Enabled bool   `mapstructure:"enabled"`
		Broker  string `mapstructure:"broker"`
	}

	Hook struct {
		c      config.Config
		conf   HookConfig
		broker Broker
		module api.Module

		sinks         []Sink
		subscriptions []Subscription
	}
)

func (h *Hook) Init(ctx context.Context, c config.Config) error {
	h.c = c

	log.Trace("loading event config...")
	if err := c.Get("event").Scan(&h.conf); err != nil {
		return err
	}
	log.Info("event config loaded",
		log.WithField("broker", h.conf.Broker),
		log.WithField("enabled", h.conf.Enabled),
	)

	// skip if disabled
	if !h.conf.Enabled {
		return nil
	}

	// load and instantiate broker
	f, err := GetBroker(h.conf.Broker)
	if err != nil {
		return err
	}

	h.module = f()
	ok := false
	if h.broker, ok = h.module.(Broker); !ok {
		return ErrEventModuleTypeInvalid
	}
	log.Trace("event broker loaded")

	if err := h.module.Init(ctx, c.Sub("event")); err != nil {
		return err
	}
	log.Trace("event broker initialized")

	// register to global broker
	globalBroker = h.broker

	return nil
}

func (h *Hook) Close(ctx context.Context) error {
	// skip if disabled
	if !h.conf.Enabled {
		return nil
	}

	log.Trace("closing subscriber...")

	// close all subscription first
	for _, sub := range h.subscriptions {
		if err := sub.Close(); err != nil {
			return err
		}
	}

	log.Trace("closing the broker...")
	// then close module
	return h.module.Close(ctx)
}

func (h *Hook) ModuleLoaded(ctx context.Context, m api.Module) {}
func (h *Hook) ModuleInitialized(ctx context.Context, m api.Module) {
	if svc, ok := m.(EventService); ok {
		h.sinks = append(h.sinks, svc.CreateSinks()...)
	}
}

func (h *Hook) Run(ctx context.Context) app.Waiter {

	for _, sink := range h.sinks {
		sub := h.broker.SubscribeHandler(ctx, sink.Topic, sink.Handler)
		if err := sub.Error(); err != nil {
			return app.NewDirectWaiter(err)
		}

		h.subscriptions = append(h.subscriptions, sub)
	}
	return app.NewDirectWaiter(nil)
}

func NewHook() *Hook {
	return &Hook{}
}
