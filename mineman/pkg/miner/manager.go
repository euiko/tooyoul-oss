package miner

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/config"
)

type (
	MiningConfig struct {
		Miner string `mapstructure:"miner"`
		Pool  string `mapstructure:"pool"`
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
	}
)

func (m *Manager) Init(ctx context.Context, c config.Config) error {
	m.c = c
	if err := m.c.Get("pools").Scan(m.pools); err != nil {
		return err
	}

	if err := m.c.Get("miner").Scan(m.minersConfig); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Close(ctx context.Context) error {
	return nil
}

func NewManager() *Manager {
	return &Manager{}
}
