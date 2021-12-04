package channel

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/event"
)

type (
	message struct {
		id           messageID
		subscriberID subscriberID
		payload      event.Payload
		broker       *Broker
	}
)

func (m *message) Scan(v interface{}, opts ...event.ScanOption) error {
	return m.payload.Scan(v, opts...)
}

func (m *message) ID() string {
	return string(m.id)
}

// Ack will acknowledge the message and release the message
func (m *message) Ack(ctx context.Context) <-chan error {
	errChan := make(chan error)
	m.broker.cmdBuffer <- ackMsgCommand{
		id:           m.id,
		subscriberID: m.subscriberID,
		ctx:          ctx,
		err:          errChan,
	}

	return errChan
}

// Progress will reserve the message for additional time
func (m *message) Progress(ctx context.Context) <-chan error {
	errChan := make(chan error)
	m.broker.cmdBuffer <- progressMsgCommand{
		id:           m.id,
		subscriberID: m.subscriberID,
		ctx:          ctx,
		err:          errChan,
	}

	return errChan
}

// Nack will reschedule the message for current subscriber
func (m *message) Nack(ctx context.Context) <-chan error {
	errChan := make(chan error)
	m.broker.cmdBuffer <- nackMsgCommand{
		id:           m.id,
		subscriberID: m.subscriberID,
		ctx:          ctx,
		err:          errChan,
	}

	return errChan
}
