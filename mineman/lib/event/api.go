package event

import "context"

// public types
type (
	EventService interface {
		CreateSinks() []Sink
	}

	Sink struct {
		Topic   string
		Handler MessageHandler
	}

	PublishConfigurator interface {
		ConfigurePublish(o *PublishOption)
	}

	PublishOptionFunc func(o *PublishOption)

	SubscribeConfigurator interface {
		ConfigureSubscribe(o *SubscribeOption)
	}

	SubscribeOptionFunc func(o *SubscribeOption)

	PublishOption struct {
	}

	SubscribeOption struct {
		Policy SubscribePolicy
	}
)

var globalBroker Broker

func (f PublishOptionFunc) ConfigurePublish(o *PublishOption) {
	f(o)
}

func (f SubscribeOptionFunc) ConfigureSubscribe(o *SubscribeOption) {
	f(o)
}

func WorkQueue() SubscribeConfigurator {
	return SubscribeOptionFunc(func(o *SubscribeOption) {
		o.Policy = WorkQueuePolicy
	})
}

func Publish(ctx context.Context, topic string, payload Payload, opts ...PublishConfigurator) error {
	// TODO: handle publish option
	if globalBroker == nil {
		return ErrEventHookNotInitialized
	}

	publishing := globalBroker.Publish(ctx, topic, payload)
	return <-publishing.Error()
}

func PublishAsync(ctx context.Context, topic string, payload Payload, opts ...PublishConfigurator) Publishing {
	// TODO: add publish option
	if globalBroker == nil {
		return NewPublishingChan(ErrEventHookNotInitialized)
	}

	return globalBroker.Publish(ctx, topic, payload)
}

func Subscribe(ctx context.Context, topic string, handler MessageHandler, opts ...SubscribeConfigurator) Subscription {
	// TODO: add subscribe option
	if globalBroker == nil {
		return NewSubscriptionDirect(ErrEventHookNotInitialized)
	}

	return globalBroker.SubscribeHandler(ctx, topic, handler)
}

func SubscribeAsync(ctx context.Context, topic string, handler MessageHandler, opts ...SubscribeConfigurator) SubscriptionMsg {
	// TODO: add subscribe option
	if globalBroker == nil {
		return NewSubscriptionDirect(ErrEventHookNotInitialized)
	}

	return globalBroker.Subscribe(ctx, topic)
}
