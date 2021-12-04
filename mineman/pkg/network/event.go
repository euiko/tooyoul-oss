package network

import (
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/event"
)

const EventStatusChangedTopic = "network.status-changed"

type (
	EventNetworkDown struct {
		At time.Time `json:"x-at"`
	}

	EventNetworkUp struct {
		At time.Time `json:"x-at"`
	}
)

func (e *EventNetworkDown) Name() string {
	return "network.down"
}

func (e *EventNetworkDown) ToEvent() *event.EventPayload {
	return &event.EventPayload{
		Name: e.Name(),
		At:   e.At,
	}
}

func (e *EventNetworkUp) Name() string {
	return "network.up"
}

func (e *EventNetworkUp) ToEvent() *event.EventPayload {
	return &event.EventPayload{
		Name: e.Name(),
		At:   e.At,
	}
}
