package channel

import (
	"context"
	"testing"

	"github.com/euiko/tooyoul/mineman/lib/event"
	"github.com/euiko/tooyoul/mineman/lib/log"
)

func TestPubSub(t *testing.T) {
	log.SetDefault(log.NewLogrusLogger())
	ctx := context.Background()
	broker := New()

	doneA := make(chan struct{})
	defer close(doneA)
	resultA := []string{}
	doneB := make(chan struct{})
	defer close(doneB)
	resultB := []string{}

	subscriberA := func(channel <-chan event.Message) {
		for {
			select {
			case msg := <-channel:
				var payload string
				msg.Scan(&payload)
				resultA = append(resultA, payload)
				msg.Ack(ctx)
			default:
				doneA <- struct{}{}
				return
			}
		}
	}

	subscriberB := func(channel <-chan event.Message) {
		for {
			select {
			case msg := <-channel:
				var payload string
				msg.Scan(&payload)
				resultB = append(resultB, payload)
				msg.Ack(ctx)
			default:
				doneB <- struct{}{}
				return
			}
		}
	}

	broker.init(ctx)
	defer broker.Close(ctx)

	subscriptionA := broker.Subscribe(ctx, "hello")
	go subscriberA(subscriptionA.Message())

	subscriptionB := broker.Subscribe(ctx, "hello")
	go subscriberB(subscriptionB.Message())

	broker.Publish(ctx, "hello", event.StringPayload("halo"))
	broker.Publish(ctx, "hello", event.StringPayload("dunia"))
	broker.Publish(ctx, "hello", event.StringPayload("apakabar"))
	broker.Publish(ctx, "hala", event.StringPayload("halo"))

	<-doneA
	<-doneB

	check := func(toCheck []string) {
		if v := toCheck[0]; v != "halo" {
			t.Fatalf("expect first result is halo, got %s", v)
			return
		}
		if v := toCheck[1]; v != "dunia" {
			t.Fatalf("expect first result is dunia, got %s", v)
			return
		}
		if v := toCheck[2]; v != "apakabar" {
			t.Fatalf("expect first result is apakabar, got %s", v)
			return
		}
	}

	check(resultA)
	check(resultB)
}
