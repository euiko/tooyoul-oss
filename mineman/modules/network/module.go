package network

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/network"
	"github.com/euiko/tooyoul/mineman/pkg/network/icmp"
)

type (
	Config struct {
		Enabled       bool          `mapstructure:"enabled"`
		Interval      time.Duration `mapstructure:"interval"`
		Count         int           `mapstructure:"count"`
		LossThreshold float64       `mapstructure:"loss_threshold"`
		DownThreshold int           `mapstructure:"down_threshold"`
		UpThreshold   int           `mapstructure:"up_threshold"`
		Targets       []string      `mapstructure:"targets"`
	}

	Module struct {
		c    config.Config
		conf Config
	}
)

func (m *Module) Init(ctx context.Context, c config.Config) error {
	m.c = c
	if err := c.Get("network").Scan(&m.conf); err != nil {
		return err
	}

	go m.runPing(ctx)

	return nil
}

func (m *Module) Close(ctx context.Context) error {
	return nil
}

func (m *Module) runPing(ctx context.Context) {

	log.Trace("running ping...")
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = m.conf.Interval

	errCount := 0
	okCount := 0

	// TODO: refactor with state machine
	for {
		select {
		case <-ctx.Done():
			return
			// re run the tests
		case <-time.After(b.NextBackOff()):

			log.Debug("doing ping...")
			if err := m.doPing(ctx); err != nil {
				if errCount > 0 && okCount > 0 {
					okCount = 0
				}

				log.Debug("do ping error", log.WithError(err))

				// when errors add the errCount
				errCount++

				// reset okCount when err threshold met
				if errCount == m.conf.DownThreshold {
					okCount = 0
					log.Debug("network changes detected to down")
					e := network.EventNetworkDown{
						At: time.Now(),
					}
					if err := event.Publish(
						ctx,
						network.EventStatusChangedTopic,
						event.FromEventDescriptor(&e),
					); err != nil {
						log.Fatal("failed when publish network status down", log.WithError(err))
					}
				}

				continue
			}

			if errCount > 0 && okCount > 0 {
				errCount = 0
			}

			// always increment ok and reset backoff when no error
			okCount++
			b.Reset()

			// reset errCount when ok threshold met
			if okCount == m.conf.UpThreshold {
				errCount = 0
				log.Debug("network changes detected to up")
				e := network.EventNetworkUp{
					At: time.Now(),
				}
				if err := event.Publish(
					ctx,
					network.EventStatusChangedTopic,
					event.FromEventDescriptor(&e),
				); err != nil {
					log.Fatal("failed when publish network status up", log.WithError(err))
				}
			}

		}
	}
}

func (m *Module) doPing(ctx context.Context) error {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		totalDone   int32 = 0
		totalTarget       = len(m.conf.Targets)
		pingChan          = make(chan icmp.PingResult)
		result            = []icmp.PingResult{}
	)

	for _, target := range m.conf.Targets {
		p, err := icmp.Ping(ctx, target, icmp.PingCount(m.conf.Count))
		if err != nil {
			return err
		}

		go func() {
			for {
				select {
				case <-p.Done():
					done := atomic.AddInt32(&totalDone, 1)
					if done >= int32(totalTarget) {
						cancel()
					}
				case res := <-p.Result():
					pingChan <- res
				}
			}
		}()
	}

collect:
	for {
		select {
		case <-newCtx.Done():
			break collect
		case res := <-pingChan:
			result = append(result, res)
		}
	}

	errorPing := []icmp.PingResult{}

	for _, r := range result {
		if r.Error() != nil {
			errorPing = append(errorPing, r)
		}
	}

	totalError := len(errorPing)
	totalPing := len(result)
	lossRatio := float64(totalError) / float64(totalPing)

	if lossRatio >= m.conf.LossThreshold {
		return errors.New(fmt.Sprintf("loss exceed threshold, with %d/%d loss detected", totalError, totalPing))
	}

	return nil
}

func New() *Module {
	return &Module{
		conf: Config{
			Interval:      time.Second * 10,
			LossThreshold: 0.2,
			Count:         3,
			DownThreshold: 2,
			UpThreshold:   2,
			Targets: []string{
				"8.8.8.8",
				"208.67.222.222",
			},
		},
	}
}

func newModule() api.Module {
	return New()
}

func init() {
	app.RegisterModule("network", newModule)
}
