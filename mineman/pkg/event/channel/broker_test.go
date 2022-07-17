package channel

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/log"
)

func TestPubSub(t *testing.T) {
	log.SetDefault(log.NewLogrusLogger())
	ctx := context.Background()
	broker := New()

	closeDone := make(chan struct{}, 1)

	doneA := make(chan struct{}, 1)
	defer close(doneA)
	resultA := []string{}
	doneB := make(chan struct{}, 1)
	defer close(doneB)
	resultB := []string{}

	subscriberA := func(sub event.SubscriptionMsg) {
		defer func() {
			doneA <- struct{}{}
		}()
		for {
			select {
			case <-sub.Done():
				return
			case msg := <-sub.Message():
				var payload string
				msg.Scan(&payload)
				resultA = append(resultA, payload)
				msg.Ack(ctx)
			}
		}
	}

	subscriberB := func(sub event.SubscriptionMsg) {
		defer func() { doneB <- struct{}{} }()
		for {
			select {
			case <-sub.Done():
				return
			case msg := <-sub.Message():
				if msg == nil {
					return
				}
				var payload string
				msg.Scan(&payload)
				resultB = append(resultB, payload)
				msg.Ack(ctx)
			}
		}
	}

	broker.Start(ctx)

	subscriptionA := broker.Subscribe(ctx, "hello")
	subscriptionB := broker.Subscribe(ctx, "hello")

	seeds := []string{
		"brown",
		"blue",
		"red",
		"green",
	}
	testCases := []string{}

	// generate test cases by power of 10
	target := int64(math.Pow10(4))
	for target > 0 {
		i := int(target % int64(len(seeds)))
		testCases = append(testCases, seeds[i])
		target--
	}

	go subscriberA(subscriptionA)
	go subscriberB(subscriptionB)

	// close the subscription after 200 milli
	go func() {
		<-time.After(time.Millisecond * 200)
		broker.Close(ctx)
		close(closeDone)
	}()

	expects := []string{}
	for _, testCase := range testCases {
		if err := <-broker.Publish(ctx, "hello", event.StringPayload(testCase)).Error(); err != nil {
			log.Trace("failed to publish with err", log.WithError(err))
			break
		}
		log.Error("hello sent")
		expects = append(expects, testCase)

		if err := <-broker.Publish(ctx, "hala", event.StringPayload(testCase)).Error(); err != nil {
			break
		}
		log.Error("hala sent")
	}

	log.Error("start function done")
	<-doneA
	log.Trace("done A")
	<-doneB
	log.Trace("done B")
	<-closeDone
	log.Trace("done all")

	check := func(toCheck []string) {
		if len(toCheck) != len(expects) {
			t.Errorf("expect result has length %d, but got %d", len(expects), len(toCheck))
			return
		}
		for i, expect := range expects {
			got := toCheck[i]
			if expect != got {
				t.Errorf("expect result on index-%d is '%s', but got '%s'", i, expect, got)
			}
		}
	}

	check(resultA)
	check(resultB)
}
