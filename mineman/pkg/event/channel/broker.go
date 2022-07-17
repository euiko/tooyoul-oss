package channel

import (
	"context"
	"errors"
	"sync"

	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/log"
)

var (
	ErrOperationCanceled       = errors.New("operation canceled")
	ErrAlreadyClosed           = errors.New("subscription already closed")
	ErrCommandBufferExceeded   = errors.New("can't handle anymore command, buffer exceeded")
	ErrPublishBufferExceeded   = errors.New("can't handle anymore publish, buffer exceeded")
	ErrSubscribeBufferExceeded = errors.New("can't send to the subscriber, buffer exceeded")
	ErrStopped                 = errors.New("channel broker stopped")
)

type (
	Config struct {
		WaitOnClose   bool `mapstructure:"wait_on_close"`
		CmdBufferSize int  `mapstructure:"cmd_buffer_size"`
		PubBufferSize int  `mapstructure:"pub_buffer_size"`
		SubBufferSize int  `mapstructure:"sub_buffer_size"`
	}

	Broker struct {
		config      Config
		ctx         context.Context
		cancel      func()
		startOnInit bool
		drainLock   sync.Mutex

		// two different channel for closer
		closeWait   chan closeCommand
		closeDirect chan closeCommand

		cmdBuffer chan command
		pubBuffer chan publishCommand

		progressMsg map[messageID]event.Message
		subsByTopic map[topicID][]subscriberID
		subs        map[subscriberID]*subscriptionChan
	}

	Options interface {
		Configure(b *Broker)
	}

	OptionsFunc func(b *Broker)

	// some alias to help for easier reading
	messageID    string
	subscriberID string
	topicID      string
)

func (f OptionsFunc) Configure(b *Broker) {
	f(b)
}

func (b *Broker) Init(ctx context.Context, c config.Config) error {
	log.Trace("loading event channel config...")
	if err := c.Get("channel").Scan(&b.config); err != nil {
		return err
	}

	if b.startOnInit {
		return <-b.Start(ctx).Wait()
	}
	return nil
}

func (b *Broker) Close(ctx context.Context) error {
	select {
	case <-b.ctx.Done():
		// do nothing
		return errors.New("context already canceled")
	default:
		if b.config.WaitOnClose {
			b.closeWait <- closeCommand{}
		} else {
			b.closeDirect <- closeCommand{}
		}
		// wait till done
		<-b.ctx.Done()
	}

	return nil
}

func (b *Broker) Publish(ctx context.Context, topic string, payload event.Payload) event.Publishing {
	errChan := make(chan error, 1)

	b.do(&publishCommand{
		topic:   topicID(topic),
		payload: payload,
		err:     errChan,
		ctx:     ctx,
	}, errChan, true)
	return event.NewPublishingChanForward(errChan)
}

func (b *Broker) Subscribe(ctx context.Context, topic string) event.SubscriptionMsg {
	subscriptionChan := make(chan event.SubscriptionMsg, 1)
	defer close(subscriptionChan)
	id := subscriberID(generateID())

	select {
	case <-b.ctx.Done():
		subscriptionChan <- newSubscriptionChan(b, errors.New("context already canceled"))
	default:
		b.cmdBuffer <- &subscribeCommand{
			ctx:          ctx,
			subscription: subscriptionChan,
			id:           id,
			topic:        topicID(topic),
		}
	}

	subscription := <-subscriptionChan

	// watch subscription cancelation to close the channel
	go func() {
		<-subscription.Done()
		subscription.Close()
	}()

	return subscription
}

func (b *Broker) SubscribeHandler(ctx context.Context, topic string, handler event.MessageHandler) event.Subscription {
	subscription := b.Subscribe(ctx, topic)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-subscription.Done():
				return
			case msg := <-subscription.Message():
				// msg nil due to closed chan, skip the loop
				if msg == nil {
					continue
				}
				handler.HandleMessage(ctx, msg)
			}
		}
	}()

	return subscription
}

func (b *Broker) run(ctx context.Context) error {
	defer log.Trace("channel broker event loop exited")

	for {
		select {
		case <-b.ctx.Done():
			log.Trace("exiting channel broker event loop...")

			// close all subs channel
			for _, v := range b.subs {
				// cancel subscription context first
				v.cancel()
				// then close the channel
				// close(v.channel)
			}
			// clear the map
			b.subs = make(map[subscriberID]*subscriptionChan)
			b.subsByTopic = make(map[topicID][]subscriberID)

			b.drainCommands()
			b.drainPublish()

			close(b.closeDirect)
			close(b.closeWait)

			return ErrStopped
		case <-b.closeDirect: // prioritize close direct command
			log.Trace("received a close direct")
			b.cancel()
		case cmd := <-b.cmdBuffer: // handle command first before publish
			log.Trace("received a command, passing to the handler...")
			b.handleCmd(b.ctx, cmd)
		case publish := <-b.pubBuffer: // separate publish with command
			log.Trace("received a publish")
			b.handlePublish(b.ctx, publish)
		case <-b.closeWait: // less prioritize the close wait command
			log.Trace("received a close wait")
			b.cancel()
		}
	}
}

func (b *Broker) do(cmd command, errChanToSend chan error, closeOnSend ...bool) error {
	select {
	case b.cmdBuffer <- cmd:
		log.Trace("cmd sent")
	default:
		errChanToSend <- ErrCommandBufferExceeded

		if len(closeOnSend) > 0 && closeOnSend[0] {
			close(errChanToSend)
		}

		return ErrCommandBufferExceeded
	}

	// drain published commands if already exited
	select {
	case <-b.ctx.Done():
		log.Trace("drain commands")
		b.drainCommands()
		b.drainPublish()
	default:
	}

	return nil
}

func (b *Broker) drainCommands() {
	b.drainLock.Lock()
	defer b.drainLock.Unlock()

	for {
		select {
		case cmd := <-b.cmdBuffer:
			b.handleCmd(b.ctx, cmd)
		default:
			// b.cmdBuffer = make(chan command, b.config.CmdBufferSize)
			return
		}
	}

}

func (b *Broker) drainPublish() {
	b.drainLock.Lock()
	defer b.drainLock.Unlock()

	for {
		select {
		case pub := <-b.pubBuffer:
			pub.err <- ErrAlreadyClosed
		default:
			// b.pubBuffer = make(chan publishCommand, b.config.PubBufferSize)
			return
		}
	}
}

func (b *Broker) Run(ctx context.Context) error {
	b.ctx, b.cancel = context.WithCancel(ctx)
	b.init()

	log.Trace("running channel broker...")
	return b.run(b.ctx)
}

func (b *Broker) Start(ctx context.Context) app.Waiter {
	b.ctx, b.cancel = context.WithCancel(ctx)
	b.init()

	log.Trace("starting channel broker...")
	go b.run(b.ctx)
	return app.NewDirectWaiter(nil)
}

func (b *Broker) init() {
	log.Trace("initializing channel broker...")
	b.closeDirect = make(chan closeCommand)
	b.closeWait = make(chan closeCommand)
	b.cmdBuffer = make(chan command, b.config.CmdBufferSize)
	b.pubBuffer = make(chan publishCommand, b.config.PubBufferSize)
	b.progressMsg = make(map[messageID]event.Message)
	b.subsByTopic = make(map[topicID][]subscriberID)
	b.subs = make(map[subscriberID]*subscriptionChan)
}

func (b *Broker) handleCmd(ctx context.Context, cmd command) {
	switch cmd := cmd.(type) {
	case *unsubscribeCommand: // handle unsubscription first
		b.handleUnsubscribe(ctx, cmd)
	case *subscribeCommand:
		b.handleSubscribe(ctx, cmd)
	case *ackMsgCommand:
		b.handleAckMsg(ctx, cmd)
	case *progressMsgCommand:
		b.handleProgressMsg(ctx, cmd)
	case *nackMsgCommand:
		b.handleNackMsg(ctx, cmd)
	case *publishCommand:
		select {
		case <-ctx.Done():
			cmd.err <- ErrAlreadyClosed
			defer close(cmd.err)
		case b.pubBuffer <- *cmd:
			// success, do nothing
		default:
			// failed to insert to publish buffer
			log.Trace("publish buffer is full")
			cmd.err <- ErrPublishBufferExceeded
			defer close(cmd.err)
		}

	}
}

func (b *Broker) handlePublish(ctx context.Context, publish publishCommand) {

	// check for a valid publish command
	if publish.err == nil {
		return
	}

	defer close(publish.err)
	// also check subscriber presence, skip all the logic if not present
	subs, ok := b.subsByTopic[publish.topic]
	if !ok {
		return
	}

	// peek all the subs not exceed buffer size
	// expect all operation must success
	for _, s := range subs {
		c := b.subs[s]
		if len(c.channel) < b.config.SubBufferSize-1 {
			continue
		}
		publish.err <- ErrSubscribeBufferExceeded
		return
	}

	// walk through all subs and send the message
	for _, s := range subs {
		// make the message for each subscriber
		id := messageID(generateID())
		msg := &message{
			id:           id,
			payload:      publish.payload,
			broker:       b,
			subscriberID: s,
		}

		// append to the processing message list
		b.progressMsg[id] = msg

		c := b.subs[s]
		c.channel <- msg
	}
}

func (b *Broker) handleUnsubscribe(ctx context.Context, unsubscribe *unsubscribeCommand) {
	// we don't respect the global ctx, because the unsubscibe also being used for
	// cleaning up resources when the context is done
	// do the clean up with reverse order to the subscribe counterparts

	select {
	case <-ctx.Done():
		unsubscribe.errChan <- ErrAlreadyClosed
		return
	case <-unsubscribe.ctx.Done():
		unsubscribe.errChan <- ErrAlreadyClosed
		return
	default:
		log.Trace("unsubscribing subscription", log.WithField("id", unsubscribe.id))
		// lookup the subscription instance
		s, ok := b.subs[unsubscribe.id]
		if !ok {
			unsubscribe.errChan <- ErrAlreadyClosed
			return
		}

		// cancel the context
		if unsubscribe.cancel != nil {
			unsubscribe.cancel()
		}

		// close the channel
		close(s.channel)

		// clear the cache by topic
		subs := b.subsByTopic[unsubscribe.topic]
		newSubs := make([]subscriberID, len(subs)-1)
		i := 0
		for _, sub := range subs {
			if sub == unsubscribe.id {
				continue
			}
			newSubs[i] = sub
			i++
		}
		b.subsByTopic[unsubscribe.topic] = newSubs

		// remove the subscription
		delete(b.subs, unsubscribe.id)

		// send the result
		unsubscribe.errChan <- nil
	}
}

func (b *Broker) handleSubscribe(ctx context.Context, subscribe *subscribeCommand) {
	// for synchronize with the main loop
	doneChan := make(chan struct{})
	defer close(doneChan)

	// do inside goroutine to also watch the context for cancel operation
	go func() {
		doChan := make(chan struct{}, 1)
		defer close(doChan)
		doChan <- struct{}{}
		closed := false

		for {
			select {
			case <-ctx.Done(): // for the global context

				if !closed {
					subscribe.subscription <- newSubscriptionChan(b, ErrOperationCanceled)
					doneChan <- struct{}{}
				}
				return
			case <-doChan: // only do when not canceled only
				// make the channel
				channel := make(chan event.Message, b.config.SubBufferSize)
				// register subscriber
				subscription := newSubscriptionChanWithChannel(
					subscribe.ctx,
					b,
					nil,
					subscribe.topic,
					subscribe.id,
					channel,
				)
				b.subs[subscribe.id] = subscription
				// cache by topic
				b.subsByTopic[subscribe.topic] = append(b.subsByTopic[subscribe.topic], subscribe.id)

				// send the result
				subscribe.subscription <- subscription

				// flag that the channel already closed
				closed = true
				doneChan <- struct{}{}
			}
		}
	}()

	// wait till registered
	<-doneChan
}

func (b *Broker) handleAckMsg(ctx context.Context, cmd *ackMsgCommand) {
	select {
	case <-ctx.Done(): // for the global context
		cmd.err <- ErrAlreadyClosed
		// close synchronization channel
		return
	case <-cmd.ctx.Done():
		cmd.err <- ErrOperationCanceled
		return
	default:
		// delete from progress message
		delete(b.progressMsg, cmd.id)
		// send result
		cmd.err <- nil
		return
	}
}

func (b *Broker) handleProgressMsg(ctx context.Context, cmd *progressMsgCommand) {
	// noop
	// TODO: add time limit for handle the message

	defer close(cmd.err)

	// send result
	cmd.err <- nil
}

func (b *Broker) handleNackMsg(ctx context.Context, cmd *nackMsgCommand) {
	defer close(cmd.err)

	// do inside goroutine to also watch the context for cancel operation
	doChan := make(chan struct{}, 1)
	defer close(doChan)
	doChan <- struct{}{}

	for {
		select {
		case <-ctx.Done(): // for the global context
			cmd.err <- ErrOperationCanceled
			// close synchronization channel
			return
		case <-cmd.ctx.Done():
			cmd.err <- ErrOperationCanceled
			return
		case <-doChan:
			// resubmit the message
			msg := b.progressMsg[cmd.id]
			c := b.subs[cmd.subscriberID]
			c.channel <- msg

			// send result
			cmd.err <- nil
			return
		}
	}
}

func WithRunOnInit(runOnInit bool) OptionsFunc {
	return OptionsFunc(func(b *Broker) {
		b.startOnInit = runOnInit
	})
}

func WithConfig(config Config) OptionsFunc {
	return OptionsFunc(func(b *Broker) {
		b.config = config
	})
}

func New(opts ...Options) *Broker {
	broker := Broker{
		startOnInit: true,
		config: Config{
			WaitOnClose:   true,
			CmdBufferSize: 256,
			PubBufferSize: 256,
			SubBufferSize: 16,
		},
	}

	for _, o := range opts {
		o.Configure(&broker)
	}

	return &broker
}

// newEventBroker return the event's Broker interface, to help
// static check whether our implementation comply with the interface
func newEventBroker() event.Broker {
	return New()
}

func newModule() api.Module {
	return newEventBroker()
}

func init() {
	event.RegisterBroker("channel", newModule)
	// also set as default broker
	event.RegisterBroker("", newModule)
}
