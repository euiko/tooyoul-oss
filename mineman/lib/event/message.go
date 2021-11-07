package event

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellh/mapstructure"
)

var (
	ErrScanEventInvalidType  = errors.New("invalid type for scan event payload, only accept event descriptor")
	ErrScanStringInvalidType = errors.New("invalid type for scan string payload, only accept string ptr")
)

type (
	Message interface {
		// Payload of the message
		Payload
		ID() string
		// Ack will acknowledge the message and release the message
		Ack(context.Context) <-chan error
		// Progress will reserve the message for additional time
		Progress(context.Context) <-chan error
		// Nack will reschedule the message for current subscriber
		Nack(context.Context) <-chan error
	}

	Payload interface {
		Scan(v interface{}) error
	}

	StringPayload string

	EventPayload struct {
		// Name refers to the what event actually happened
		Name string
		// At represent time when the event occurred
		At time.Time
		// Data hold the event supportive data
		Data map[string]interface{}
		// Meta hold data that help to describe/distinguish event with the others
		// e.g user, tenant, etc.
		Meta map[string]interface{}
	}

	EventDescriptor interface{}
)

// Scan on event payload means unmarshal an event to appropriate event descriptor
func (p StringPayload) Scan(v interface{}) error {
	str, ok := v.(*string)
	if !ok {
		return ErrScanStringInvalidType
	}

	*str = string(p)
	return nil
}

// Scan on event payload means unmarshal an event to appropriate event descriptor
func (p *EventPayload) Scan(v interface{}) error {
	_, ok := v.(EventDescriptor)
	if !ok {
		return ErrScanEventInvalidType
	}

	// add any event data with x prefix
	data := map[string]interface{}{
		"x-name": p.Name,
		"x-at":   p.At,
	}

	// add meta data with x-meta prefix
	for k, v := range p.Meta {
		data["x-meta-"+k] = v
	}

	// add data payload
	for k, v := range p.Data {
		data[k] = v
	}

	return mapstructure.Decode(data, v)
}
