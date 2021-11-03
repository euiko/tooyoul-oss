package channel

import (
	"context"

	"github.com/euiko/tooyoul/mineman/lib/event"
)

type (
	command interface{}

	publishCommand struct {
		ctx     context.Context
		topic   topicID
		payload event.Payload
		err     chan error
	}

	subscribeCommand struct {
		ctx          context.Context
		subscription chan event.SubscriptionMsg
		id           subscriberID
		topic        topicID
	}

	unsubscribeCommand struct {
		id      subscriberID
		topic   topicID
		errChan chan error
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
