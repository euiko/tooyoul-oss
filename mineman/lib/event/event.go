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

	Subscriber interface {
		Subscribe(ctx context.Context, topic string) SubscriptionMsg
		SubscribeHandler(ctx context.Context, topic string, handler MessageHandler) Subscription
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
		Done() <-chan struct{}
		Message() <-chan Message
	}
)

// private type
type (
	publishingChan struct {
		manageClose bool
		channel     chan error
	}

	subscriptionDirect struct {
		err         error
		closer      func() error
		doneChan    chan struct{}
		messageChan chan Message
	}
)

func (p *publishingChan) Error() <-chan error {
	if p.manageClose {
		defer close(p.channel)
	}

	return p.channel

}
func NewPublishingChan(initial error) Publishing {
	errChan := make(chan error)

	if initial != nil {
		errChan <- initial
	}

	return &publishingChan{
		manageClose: true,
		channel:     errChan,
	}
}

func NewPublishingChanForward(channel chan error) Publishing {
	return &publishingChan{
		manageClose: false,
		channel:     channel,
	}
}

func (s *subscriptionDirect) Error() error {
	return s.err
}

func (s *subscriptionDirect) Close() error {
	if s.closer == nil {
		return nil
	}

	return s.closer()
}

func (s *subscriptionDirect) Message() <-chan Message {
	return s.messageChan
}

func (s *subscriptionDirect) Done() <-chan struct{} {
	return s.doneChan
}

// NewSubscriptionDirect emulate subscription that will be
// closed directly upon first listening
func NewSubscriptionDirect(initial error) SubscriptionMsg {
	doneChan := make(chan struct{})
	msgChan := make(chan Message)

	// close directly after its creation
	defer close(doneChan)
	defer close(msgChan)

	doneChan <- struct{}{}
	return &subscriptionDirect{
		err:         initial,
		closer:      nil,
		doneChan:    doneChan,
		messageChan: msgChan,
	}
}

// NewSubscriptionForward will forward supplied parameters and
// wrap it to compatible wirh subscription interface
func NewSubscriptionForward(
	initial error,
	closer func() error,
	doneChan chan struct{},
	messageChan chan Message,
) SubscriptionMsg {
	return &subscriptionDirect{
		err:         initial,
		closer:      closer,
		doneChan:    doneChan,
		messageChan: messageChan,
	}
}

func (h MessageHandlerFunc) HandleMessage(ctx context.Context, message Message) {
	h(ctx, message)
}
