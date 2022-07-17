package miner

import (
	"context"
	"fmt"

	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/log"
)

type (
	MiningConfig struct {
		Miner  string `mapstructure:"miner"`
		Pool   string `mapstructure:"pool"`
		Device string `mapstructure:"device"`
	}
	Pool struct {
		Url       string    `mapstructure:"url"`
		User      string    `mapstructure:"user"`
		Pass      string    `mapstructure:"pass"`
		Algorithm Algorithm `mapstructure:"algorithm"`
	}
	Manager struct {
		c            config.Config
		pools        map[string]Pool
		minersConfig []MiningConfig
		miners       []Miner
	}
)

func (m *Manager) Init(ctx context.Context, c config.Config) error {
	m.c = c
	if err := m.c.Get("pools").Scan(&m.pools); err != nil {
		return err
	}

	if err := m.c.Get("miners").Scan(&m.minersConfig); err != nil {
		return err
	}

	m.miners = make([]Miner, len(m.minersConfig))
	for i, config := range m.minersConfig {
		configKey := fmt.Sprintf("miners.%d", i)
		miner, err := m.createMiner(ctx, m.c.Sub(configKey), config)
		if err != nil {
			return err
		}
		m.miners[i] = miner
	}

	return nil
}

func (m *Manager) Close(ctx context.Context) error {

	if err := m.Stop(ctx); err != nil {
		return err
	}

	for _, miner := range m.miners {
		if err := miner.Close(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	log.Trace("starting miner...")

	for _, miner := range m.miners {
		if err := miner.Start(ctx); err != nil {
			if err == ErrMinerAlreadyStarted {
				log.Trace("start miner %s skipped, it is already started", log.WithValues(miner.Name()))
				continue
			}

			return err
		}
	}

	log.Trace("miner started")
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	log.Trace("stopping miner...")

	for _, miner := range m.miners {
		if err := miner.Stop(); err != nil {
			if err == ErrMinerAlreadyStopped {
				log.Trace("stop miner %s skipped, it is already stopped", log.WithValues(miner.Name()))
				continue
			}

			return err
		}
	}

	log.Trace("miner stopped")
	return nil
}

func (m *Manager) createMiner(ctx context.Context, c config.Config, config MiningConfig) (Miner, error) {

	pool, ok := m.pools[config.Pool]
	if !ok {
		return nil, fmt.Errorf("pool %s doesn't exists, are your forget to add the pools", config.Pool)
	}

	deviceQuery, err := ParseDeviceQuery(config.Device)
	if err != nil {
		return nil, err
	}

	// first use default path executor
	executor := NewPathExecutor("")

	factory, ok := globalRegistry[config.Miner]
	if !ok {
		return nil, fmt.Errorf("miner %s doesn't exists, make sure you are using supported miner", config.Miner)
	}

	settings := newSettings(
		WithPool(pool),
		WithDevice(deviceQuery),
		WithExecutor(executor),
	)
	miner := factory(settings)
	if !miner.Available() {
		// TODO: handle when miner program not available (maybe download from source)
		return nil, fmt.Errorf("miner %s are not available in your system, make sure you are install it properly", config.Miner)
	}
	if err := miner.Init(ctx, c); err != nil {
		return nil, err
	}

	return miner, nil
}

func NewManager() *Manager {
	return &Manager{}
}
