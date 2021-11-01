package channel

import (
	"errors"

	"github.com/euiko/tooyoul/mineman/lib/event"
)

type (
	subscriptionChan struct {
		channel chan event.Message
	}
)

func (s *subscriptionChan) Close() error {
	return errors.New("not yet implemented")
}

func (s *subscriptionChan) Message() <-chan event.Message {
	return s.channel
}
