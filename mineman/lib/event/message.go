package event

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellh/mapstructure"
)

var (
	ErrScanEventInvalidType  = errors.New("invalid type for scan event payload, only accept event descriptor")
	ErrScanEventNameNotMatch = errors.New("invalid event name for scan event payload, only can marshal with matched name")
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

	scanOption struct {
		strict bool
	}

	ScanOption interface {
		ConfigureScan(o *scanOption)
	}

	ScanOptionFunc func(o *scanOption)

	Payload interface {
		Scan(v interface{}, opts ...ScanOption) error
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

	EventDescriptor interface {
		Name() string
		ToEvent() *EventPayload
	}
)

func (f ScanOptionFunc) ConfigureScan(o *scanOption) {
	f(o)
}

// Scan on event payload means unmarshal an event to appropriate event descriptor
func (p StringPayload) Scan(v interface{}, opts ...ScanOption) error {
	str, ok := v.(*string)
	if !ok {
		return ErrScanStringInvalidType
	}

	*str = string(p)
	return nil
}

// Scan on event payload means unmarshal an event to appropriate event descriptor
func (p *EventPayload) Scan(v interface{}, opts ...ScanOption) error {
	opt := loadScanOption(opts...)

	ed, ok := v.(EventDescriptor)
	if !ok {
		return ErrScanEventInvalidType
	}

	// compare the event name with the descriptor, when strict mode
	if opt.strict && ed.Name() != p.Name {
		return ErrScanEventNameNotMatch
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

func ScanStrictMode(strictMode bool) ScanOption {
	return ScanOptionFunc(func(o *scanOption) {
		o.strict = strictMode
	})
}

func FromEventDescriptor(ed EventDescriptor) Payload {
	return ed.ToEvent()
}

func loadScanOption(opts ...ScanOption) *scanOption {
	o := scanOption{
		strict: true,
	}

	for _, f := range opts {
		f.ConfigureScan(&o)
	}

	return &o
}
