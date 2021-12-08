package miner

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/miner"
	"github.com/euiko/tooyoul/mineman/pkg/network"

	_ "github.com/euiko/tooyoul/mineman/pkg/miner/teamredminer"
)

type (
	Module struct {
		c       config.Config
		manager *miner.Manager
		ctx     context.Context
	}
)

func (m *Module) Init(ctx context.Context, c config.Config) error {
	m.c = c
	m.manager = miner.NewManager()
	m.ctx = context.Background()

	if err := m.manager.Init(ctx, c.Sub("miner")); err != nil {
		return err
	}

	return nil
}

func (m *Module) Close(ctx context.Context) error {
	return m.manager.Close(ctx)
}

func (m *Module) CreateEndpoints(mws ...api.Middleware) []api.Endpoint {
	return []api.Endpoint{}
}

func (m *Module) CreateSinks() []event.Sink {
	return []event.Sink{
		{
			Topic:   network.EventStatusChangedTopic,
			Handler: m.networkChangedEventHandler(),
		},
	}
}

func New() *Module {
	return &Module{}
}

func newApiModule() api.Module {
	return New()
}

func init() {
	app.RegisterModule("miner", newApiModule)
}
