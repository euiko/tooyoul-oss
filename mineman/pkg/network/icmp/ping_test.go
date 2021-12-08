package icmp

import (
	"context"
	"testing"
)

func TestPing(t *testing.T) {
	const totalPing = 20

	ctx := context.Background()
	p, err := Ping(ctx, "google.com", PingCount(totalPing))
	if err != nil {
		t.Fatal("error when create a ping instance", err)
		return
	}

	successResult := 0
l:
	for {
		select {
		case <-p.Done():
			break l
		case res := <-p.Result():
			if err := res.Error(); res.IsOk() && err != nil {
				t.Log("failed when pinging with error", err)
				continue
			}
			successResult++
			t.Logf("ping seq %d success", res.Sequence)
		}
	}

	if successResult != totalPing {
		t.Fatalf("expect successResult is %d, but got %d", totalPing, successResult)
		return
	}
}
