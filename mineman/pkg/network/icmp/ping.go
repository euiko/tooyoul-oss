package icmp

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/network"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	ErrPingUnknown = errors.New("unknown error when pinging")
)

const (
	PingPayload       = "HELLO-R-U-THERE"
	PingMaxReadBuffer = 1500
)

type (
	PingRequest struct {
		count    int
		target   string
		targetIP net.IP
		interval time.Duration
	}

	PingResult struct {
		Source   string
		SourceIP net.IP
		PeerAddr net.Addr
		Type     *ipv4.ICMPType
		Sequence int
		err      error
	}

	PingOption func(p *PingRequest)

	Pinging struct {
		channel  chan PingResult
		doneChan chan struct{}
	}
)

func (p *Pinging) Done() <-chan struct{} {
	return p.doneChan
}
func (p *Pinging) Result() <-chan PingResult {
	return p.channel
}

func (r *PingResult) IsOk() bool {
	if r.Type != nil && *r.Type == ipv4.ICMPTypeEchoReply {
		return true
	}
	return false
}

func (r *PingResult) Error() error {
	if r.err != nil {
		return r.err
	} else if r.Type != nil && *r.Type != ipv4.ICMPTypeEchoReply {
		errMsg := r.Type.String()
		return errors.New(errMsg)
	}

	return nil
}

func PingInfinity() PingOption {
	return func(p *PingRequest) {
		p.count = -1
	}
}

func PingCount(count int) PingOption {
	return func(p *PingRequest) {
		p.count = count
	}
}

func Ping(ctx context.Context, target string, opts ...PingOption) (*Pinging, error) {
	var (
		err error
	)

	req := PingRequest{
		target:   target,
		count:    3,
		interval: time.Second,
	}
	for _, opt := range opts {
		opt(&req)
	}
	log.Trace("ping requested",
		log.WithField("target", req.target),
		log.WithField("count", req.count),
	)

	// resolv target ip
	req.targetIP, err = network.ResolveIP(req.target)
	if err != nil {
		return nil, err
	}
	log.Trace("ping target ip resolved", log.WithField("ip", req.targetIP))

	pinging := Pinging{
		channel:  make(chan PingResult),
		doneChan: make(chan struct{}),
	}

	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal("failed when listening icmp packet", log.WithError(err))
		close(pinging.channel)
		close(pinging.doneChan)

		return nil, err
	}

	go runPing(ctx, c, &req, &pinging)

	return &pinging, nil
}

func newPingResultFromReq(seq int, req *PingRequest) PingResult {
	return PingResult{
		Source:   req.target,
		SourceIP: req.targetIP,
		Sequence: seq,
	}
}

func runPing(ctx context.Context, conn *icmp.PacketConn, req *PingRequest, pinging *Pinging) {
	defer close(pinging.channel)
	defer close(pinging.doneChan)
	defer conn.Close()

	seq := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-pinging.Done():
			return
		case <-time.After(req.interval):
			if req.count > 0 && seq <= req.count {
				doPing(seq, conn, req, pinging)
				seq++
				continue
			}

			pinging.doneChan <- struct{}{}
		}
	}
}

func doPing(seq int, conn *icmp.PacketConn, req *PingRequest, pinging *Pinging) {
	result := newPingResultFromReq(seq, req)

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seq,
			Data: []byte(PingPayload),
		},
	}

	// make write buffer and write to the connection
	wb, err := msg.Marshal(nil)
	if err != nil {
		log.Fatal("failed when marshalling icmp message", log.WithError(err))
		result.err = err
		pinging.channel <- result
		return
	}
	targetStr := req.targetIP.String()
	if _, err := conn.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(targetStr)}); err != nil {
		log.Debug("ping write failed", log.WithError(err))
		result.err = err
		pinging.channel <- result
		return
	}

	// make the read buffer and read from the connection
	rb := make([]byte, PingMaxReadBuffer)
	n, peer, err := conn.ReadFrom(rb)
	if err != nil {
		log.Debug("ping read failed", log.WithError(err))
		result.err = err
		pinging.channel <- result
		return
	}

	result.PeerAddr = peer

	// read as message
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:n])
	if err != nil {
		log.Debug("ping parse message failed", log.WithError(err))
		result.err = err
		pinging.channel <- result
		return
	}

	// try convert and read the icmp type
	icmpType, ok := rm.Type.(ipv4.ICMPType)
	if !ok {
		log.Debug("ping obtain result type failed", log.WithError(err))
		result.err = err
		pinging.channel <- result
		return
	}

	result.Type = &icmpType
	pinging.channel <- result
}
