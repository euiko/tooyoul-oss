package channel

import (
	"github.com/euiko/tooyoul/mineman/lib/event"
)

type (
	subscriptionChan struct {
		broker  *Broker
		id      subscriberID
		topic   topicID
		err     error
		channel chan event.Message
	}
)

func (s *subscriptionChan) Close() error {
	errChan := make(chan error)
	defer close(errChan)

	s.broker.cmdBuffer <- &unsubscribeCommand{
		id:      s.id,
		topic:   s.topic,
		errChan: errChan,
	}

	return <-errChan
}

func (s *subscriptionChan) Message() <-chan event.Message {
	return s.channel
}

func (s *subscriptionChan) Error() error {
	return s.err
}
