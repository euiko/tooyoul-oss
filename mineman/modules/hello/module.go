package hello

import (
	"context"
	"net/http"

	"github.com/euiko/tooyoul/mineman/pkg/app"
	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/event"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/network"
)

type Module struct {
}

func (m *Module) Init(ctx context.Context, c config.Config) error {
	event.Subscribe(ctx, network.EventStatusChangedTopic, event.MessageHandlerFunc(func(ctx context.Context, message event.Message) {
		var (
			downEvent network.EventNetworkDown
			upEvent   network.EventNetworkDown
		)

		if err := message.Scan(&downEvent); err == nil {
			log.Info("network is down", log.WithField("time", downEvent.At))
		} else if err := message.Scan(&upEvent); err == nil {
			log.Info("network is up", log.WithField("time", upEvent.At))
		}

		message.Ack(ctx)
	}))
	return nil
}

func (m *Module) Close(ctx context.Context) error {
	return nil
}

func (m *Module) CreateEndpoints(mws ...api.Middleware) []api.Endpoint {
	return []api.Endpoint{
		{
			Method:  "GET",
			Path:    "/hello",
			Handler: m.helloHandler(),
		},
	}
}

func (m *Module) helloHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("hello")
		w.Write([]byte("hello"))
	})
}

func NewModule() api.Module {
	return &Module{}
}

func init() {
	app.RegisterModule("hello", NewModule)
}
