package api

import (
	"context"

	"github.com/euiko/tooyoul/mineman/pkg/config"
)

type (
	Module interface {
		Init(ctx context.Context, config config.Config) error
		Close(ctx context.Context) error
	}
)
