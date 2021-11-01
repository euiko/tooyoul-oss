package channel

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/event"
)

type (
	command interface{}

	publishCommand struct {
		topic   topicID
		payload event.Payload
		err     chan error
	}

	subscribeCommand struct {
		subscription chan event.SubscriptionMsg
		id           subscriberID
		topic        topicID
		channel      chan event.Message
	}

	ackMsgCommand struct {
		id           messageID
		subscriberID subscriberID
		ctx          context.Context
		err          chan error
	}

	nackMsgCommand struct {
		id           messageID
		subscriberID subscriberID
		ctx          context.Context
		err          chan error
	}

	progressMsgCommand struct {
		id           messageID
		subscriberID subscriberID
		ctx          context.Context
		err          chan error
	}

	closeCommand struct{}
)
