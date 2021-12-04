package miner

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
)

type (
	Module struct {
		c config.Config
	}
)

func (m *Module) Init(ctx context.Context, c config.Config) error {
	return nil
}

func (m *Module) Close(ctx context.Context) error {
	return nil
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
