package channel

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/event"
)

type (
	subscriptionChan struct {
		broker  *Broker
		id      subscriberID
		topic   topicID
		err     error
		channel chan event.Message

		// to hold cancelation with ease
		ctx    context.Context
		cancel func()
	}
)

func (s *subscriptionChan) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *subscriptionChan) Close() error {
	errChan := make(chan error)

	s.broker.cmdBuffer <- &unsubscribeCommand{
		id:      s.id,
		topic:   s.topic,
		errChan: errChan,
		cancel:  s.cancel,
	}

	return <-errChan
}

func (s *subscriptionChan) Message() <-chan event.Message {
	return s.channel
}

func (s *subscriptionChan) Error() error {
	return s.err
}

func newSubscriptionChan(b *Broker, err error) *subscriptionChan {
	return newSubscriptionChanWithChannel(context.Background(), b, err, "", "", nil)
}

func newSubscriptionChanWithChannel(
	ctx context.Context,
	b *Broker,
	err error,
	topic topicID,
	subID subscriberID,
	channel chan event.Message,
) *subscriptionChan {
	ctx, cancel := context.WithCancel(ctx)
	return &subscriptionChan{
		ctx:     ctx,
		cancel:  cancel,
		broker:  b,
		err:     err,
		topic:   topic,
		id:      subID,
		channel: channel,
	}
}
