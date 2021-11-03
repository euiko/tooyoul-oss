package event

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
)

const (
	WorkQueuePolicy SubscribePolicy = iota
)

type (
	SubscribePolicy int

	Broker interface {
		api.Module
		Subscriber
		Publisher
	}

	Publisher interface {
		Publish(ctx context.Context, topic string, payload Payload) Publishing
	}

	MessageHandler interface {
		HandleMessage(ctx context.Context, message Message)
	}

	MessageHandlerFunc func(ctx context.Context, message Message)

	SubscribeOption struct {
		Policy SubscribePolicy
	}

	SubscribeOptionFunc func(o *SubscribeOption)

	Subscriber interface {
		Subscribe(ctx context.Context, topic string) SubscriptionMsg
		SubscribeMessage(ctx context.Context, topic string, handle MessageHandler) Subscription
	}

	Publishing interface {
		Error() <-chan error
	}

	Subscription interface {
		Error() error
		Close() error
	}

	SubscriptionMsg interface {
		Subscription
		Message() <-chan Message
	}
)

func WorkQueue() SubscribeOptionFunc {
	return func(o *SubscribeOption) {
		o.Policy = WorkQueuePolicy
	}
}

func (h MessageHandlerFunc) HandleMessage(ctx context.Context, message Message) {
	h(ctx, message)
}
