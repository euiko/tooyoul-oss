package channel

import (
	"context"
	"errors"

	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/event"
	"github.com/euiko/tooyoul/mineman/lib/log"
)

var (
	ErrPublishBufferExceeded   = errors.New("can't handle anymore publish, buffer exceeded")
	ErrSubscribeBufferExceeded = errors.New("can't send to the subscriber, buffer exceeded")
)

type (
	Config struct {
		WaitOnClose   bool `mapstructure:"wait_on_close"`
		CmdBufferSize int  `mapstructure:"cmd_buffer_size"`
		PubBufferSize int  `mapstructure:"pub_buffer_size"`
		SubBufferSize int  `mapstructure:"sub_buffer_size"`
	}

	Broker struct {
		config Config
		cancel func()

		// two different channel for closer
		closeWait   chan closeCommand
		closeDirect chan closeCommand

		cmdBuffer chan command
		pubBuffer chan publishCommand

		progressMsg map[messageID]event.Message
		subsByTopic map[topicID][]subscriberID
		subs        map[subscriberID]chan event.Message
	}

	publishing struct {
		channel <-chan error
	}

	// some alias to help for easier reading
	messageID    string
	subscriberID string
	topicID      string
)

func (b *Broker) Init(ctx context.Context, c config.Config) error {
	log.Trace("loading event channel config...")
	if err := c.Get("event.channel").Scan(&b.config); err != nil {
		return err
	}

	b.init(ctx)
	return nil
}

func (b *Broker) Close(ctx context.Context) error {

	if b.config.WaitOnClose {
		b.closeWait <- closeCommand{}
	} else {
		b.closeDirect <- closeCommand{}
	}

	close(b.closeDirect)
	close(b.closeWait)
	close(b.cmdBuffer)
	close(b.pubBuffer)
	for _, v := range b.subs {
		close(v)
	}

	return nil
}

func (b *Broker) Publish(ctx context.Context, topic string, payload event.Payload) event.Publishing {
	errChan := make(chan error)
	b.cmdBuffer <- &publishCommand{
		topic:   topicID(topic),
		payload: payload,
		err:     errChan,
	}
	return &publishing{
		channel: errChan,
	}
}

func (b *Broker) Subscribe(ctx context.Context, topic string) event.SubscriptionMsg {
	subscriptionChan := make(chan event.SubscriptionMsg)
	defer close(subscriptionChan)
	id := subscriberID(generateID())

	b.cmdBuffer <- &subscribeCommand{
		subscription: subscriptionChan,
		id:           id,
		topic:        topicID(topic),
		channel:      make(chan event.Message, b.config.SubBufferSize),
	}
	return <-subscriptionChan
}

func (b *Broker) SubscribeCallback(ctx context.Context, topic string) event.Subscription {
	panic("not yet implemented")
}

func (b *Broker) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.closeDirect: // prioritize close direct command
			b.cancel()
		case cmd := <-b.cmdBuffer: // handle command first before publish
			b.handleCmd(ctx, cmd)
		case publish := <-b.pubBuffer: // separate publish with command
			b.handlePublish(ctx, publish)
		case <-b.closeWait: // less prioritize the close wait command
			b.cancel()
		}
	}
}

func (b *Broker) init(ctx context.Context) {
	ctx, b.cancel = context.WithCancel(ctx)

	log.Trace("initializing channel broker...")
	b.closeDirect = make(chan closeCommand)
	b.closeWait = make(chan closeCommand)
	b.cmdBuffer = make(chan command, b.config.CmdBufferSize)
	b.pubBuffer = make(chan publishCommand, b.config.PubBufferSize)
	b.progressMsg = make(map[messageID]event.Message)
	b.subsByTopic = make(map[topicID][]subscriberID)
	b.subs = make(map[subscriberID]chan event.Message)

	log.Trace("starting channel broker...")
	go b.start(ctx)
}

func (b *Broker) handleCmd(ctx context.Context, cmd command) {
	switch cmd.(type) {
	case *publishCommand:
		publish := cmd.(*publishCommand)
		select {
		case b.pubBuffer <- *publish:
			// success, do nothing
		default:
			// failed to insert to publish buffer
			publish.err <- ErrPublishBufferExceeded
			defer close(publish.err)
		}
	case *subscribeCommand:
		b.handleSubscribe(ctx, cmd.(*subscribeCommand))
	case *ackMsgCommand:
		b.handleAckMsg(ctx, cmd.(*ackMsgCommand))
	case *progressMsgCommand:
		b.handleProgressMsg(ctx, cmd.(*progressMsgCommand))
	case *nackMsgCommand:
		b.handleNackMsg(ctx, cmd.(*nackMsgCommand))

	}
}

func (b *Broker) handlePublish(ctx context.Context, publish publishCommand) {
	subs := b.subsByTopic[publish.topic]
	defer close(publish.err)

	// peek all the subs not exceed buffer size
	// expect all operation must success
	for _, s := range subs {
		c := b.subs[s]
		if len(c) < b.config.SubBufferSize-1 {
			continue
		}
		publish.err <- ErrSubscribeBufferExceeded
		break
	}

	// walk through all subs and send the message
	for _, s := range subs {
		// make the message for each subscriber
		id := messageID(generateID())
		msg := &message{
			id:      id,
			payload: publish.payload,
			broker:  b,
		}

		// append to the processing message list
		b.progressMsg[id] = msg

		c := b.subs[s]
		c <- msg
	}
}

func (b *Broker) handleSubscribe(ctx context.Context, subscribe *subscribeCommand) {
	// register subscriber
	b.subs[subscribe.id] = subscribe.channel
	// cache by topic
	b.subsByTopic[subscribe.topic] = append(b.subsByTopic[subscribe.topic], subscribe.id)
	// send the result
	subscribe.subscription <- &subscriptionChan{
		channel: subscribe.channel,
	}
}

func (b *Broker) handleAckMsg(ctx context.Context, cmd *ackMsgCommand) {
	// delete from progress message
	delete(b.progressMsg, cmd.id)
}

func (b *Broker) handleProgressMsg(ctx context.Context, cmd *progressMsgCommand) {
	// noop
	// TODO: add time limit for handle the message
}

func (b *Broker) handleNackMsg(ctx context.Context, cmd *nackMsgCommand) {
	// resubmit the message
	msg := b.progressMsg[cmd.id]
	c := b.subs[cmd.subscriberID]
	c <- msg
}

func New() *Broker {
	return &Broker{
		config: Config{
			WaitOnClose:   false,
			CmdBufferSize: 256,
			PubBufferSize: 256,
			SubBufferSize: 16,
		},
	}
}

func (p *publishing) Error() <-chan error {
	return p.channel
}