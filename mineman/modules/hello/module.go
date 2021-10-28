package hello

import (
	"context"
	"net/http"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/logger"
)

type Module struct {
}

func (m *Module) Init(ctx context.Context, c config.Config) error {
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
		logger.Info(r.Context(), logger.Message("hello"))
		w.Write([]byte("hello"))
	})
}
