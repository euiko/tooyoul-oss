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

	// DefaultModule is extension to Module that specify whether is default loaded
	// TODO: consider more proper interface that can be customized its behaviour
	// inside the framework
	DefaultModule interface {
		Default() bool
	}
)
