package miner

import (
	"context"
	"errors"

	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/network"
)

func (m *Module) networkChangedEventHandler() event.MessageHandler {
	return event.MessageHandlerFuncErr(func(ctx context.Context, message event.Message) error {
		var (
			networkUpEvent   network.EventNetworkUp
			networkDownEvent network.EventNetworkDown
			err              error
		)

		if err = message.Scan(&networkUpEvent); err == nil {
			return m.manager.Start(m.ctx)
		} else if err = message.Scan(&networkDownEvent); err == nil {
			return m.manager.Stop(m.ctx)
		}

		return errors.New("event are not handled")
	})
}
