package app

import (
	"context"
	"net/http"
	"time"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
	"github.com/euiko/tooyoul/mineman/lib/config"
	"github.com/euiko/tooyoul/mineman/lib/logger"
	"github.com/julienschmidt/httprouter"
)

type (
	WebOption struct {
		Enabled      bool          `mapstructure:"enabled"`
		Address      string        `mapstructure:"address"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	}
	WebHook struct {
		option    WebOption
		c         config.Config
		server    http.Server
		errChan   chan error
		endpoints []api.Endpoint
		defaultMw []api.Middleware
	}
)

func (h *WebHook) Init(ctx context.Context, c config.Config) error {
	h.c = c
	logger.Debug(ctx, logger.Message(c.Get("wen.enabled").Bool()))

	if err := h.c.Get("web").Scan(&h.option); err != nil {
		return err
	}

	if !h.option.Enabled {
		return nil
	}

	h.server.Addr = h.option.Address
	h.server.ReadTimeout = h.option.ReadTimeout
	h.server.WriteTimeout = h.option.WriteTimeout

	return nil
}

func (h *WebHook) Close(ctx context.Context) error {
	if !h.option.Enabled {
		return nil
	}

	return h.stop(ctx)
}

func (h *WebHook) ModuleLoaded(ctx context.Context, m api.Module) {}
func (h *WebHook) ModuleInitialized(ctx context.Context, m api.Module) {
	if svc, ok := m.(api.WebService); ok {
		h.endpoints = append(h.endpoints, svc.CreateEndpoints()...)
	}
}

func (h *WebHook) Run(ctx context.Context) Waiter {
	if !h.option.Enabled {
		return nil
	}

	router := httprouter.New()
	defaultMiddlerwares := []api.Middleware{loggerInjectorMiddleware(ctx)}

	for _, endpoint := range h.endpoints {

		var skipMiddlewares bool
		if ext, ok := endpoint.Handler.(api.SkipMiddlewaresExt); ok {
			skipMiddlewares = ext.SkipMiddlewares()
		}

		var skipDefaultMiddlewares bool
		if ext, ok := endpoint.Handler.(api.SkipDefaultMiddlewaresExt); ok {
			skipDefaultMiddlewares = ext.SkipDefaultMiddlewares()
		}

		effectiveMiddlewares := defaultMiddlerwares

		if !skipDefaultMiddlewares {
			effectiveMiddlewares = append(effectiveMiddlewares, h.defaultMw...)
		}

		if skipMiddlewares {
			effectiveMiddlewares = defaultMiddlerwares
		}

		router.Handler(endpoint.Method, endpoint.Path, h.handleWithMiddlewares(endpoint.Handler, effectiveMiddlewares...))
	}

	h.server.Handler = router
	go h.start(ctx)
	return NewChanWaiter(h.errChan)
}

func (h *WebHook) handleWithMiddlewares(handler http.Handler, mws ...api.Middleware) http.Handler {
	if len(mws) < 1 {
		return handler
	}

	return h.handleWithMiddlewares(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mws[0].Handle(handler).ServeHTTP(w, r)
	}), mws[1:]...)
}

func (h *WebHook) start(ctx context.Context) {
	logger.Trace(ctx, logger.Message("starting web service..."))
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(ctx, logger.Message("failed when listen and serve http err=", err))
		h.errChan <- err
	} else {
		h.errChan <- nil
	}
	close(h.errChan)
}

func (h *WebHook) stop(ctx context.Context) error {
	logger.Trace(ctx, logger.Message("stopping web service..."))
	err := h.server.Close()
	logger.Trace(ctx, logger.Message("web service stopped"))
	return err
}

func NewWebHook() *WebHook {
	return &WebHook{
		errChan: make(chan error),
	}
}

func loggerInjectorMiddleware(ctx context.Context) api.Middleware {
	l := logger.FromContext(ctx)
	return api.MiddlewareFunc(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rCtx := r.Context()
			injectedReq := r.WithContext(logger.InjectContext(rCtx, l))
			h.ServeHTTP(w, injectedReq)
		})
	})
}
