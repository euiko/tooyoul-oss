package app

import (
	"context"
	"net/http"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/app/api"
	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/julienschmidt/httprouter"
)

type (
	WebConfig struct {
		Enabled      bool          `mapstructure:"enabled"`
		Address      string        `mapstructure:"address"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	}
	WebHook struct {
		option    WebConfig
		c         config.Config
		server    http.Server
		errChan   chan error
		endpoints []api.Endpoint
		defaultMw []api.Middleware
	}
)

func (h *WebHook) Init(ctx context.Context, c config.Config) error {

	// load config
	h.c = c
	if err := h.c.Get("web").Scan(&h.option); err != nil {
		return err
	}

	// skip if disabled
	if !h.option.Enabled {
		return nil
	}

	// set some server parameters
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

func (h *WebHook) Run(ctx context.Context) error {

	// skip if disabled
	if !h.option.Enabled {
		return nil
	}

	// create new router
	router := httprouter.New()

	// defaultMiddlewares are absolute middleware to be used even if all the middleware skipped
	defaultMiddlerwares := []api.Middleware{loggerInjectorMiddleware(ctx)}

	for _, endpoint := range h.endpoints {

		// skip all middleware
		var skipMiddlewares bool
		if ext, ok := endpoint.Handler.(api.SkipMiddlewaresExt); ok {
			skipMiddlewares = ext.SkipMiddlewares()
		}

		// skip the default middleware provided by the web hook
		var skipDefaultMiddlewares bool
		if ext, ok := endpoint.Handler.(api.SkipDefaultMiddlewaresExt); ok {
			skipDefaultMiddlewares = ext.SkipDefaultMiddlewares()
		}

		// effectiveMiddlewares is the actual middleware to be used by an endpoint
		effectiveMiddlewares := defaultMiddlerwares

		if !skipDefaultMiddlewares {
			effectiveMiddlewares = append(effectiveMiddlewares, h.defaultMw...)
		}

		if skipMiddlewares {
			effectiveMiddlewares = defaultMiddlerwares
		}

		// add to the router
		router.Handler(endpoint.Method, endpoint.Path, h.handleWithMiddlewares(endpoint.Handler, effectiveMiddlewares...))
	}

	h.server.Handler = router
	return h.start(ctx)
}

func (h *WebHook) handleWithMiddlewares(handler http.Handler, mws ...api.Middleware) http.Handler {
	if len(mws) < 1 {
		return handler
	}

	return h.handleWithMiddlewares(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mws[0].Handle(handler).ServeHTTP(w, r)
	}), mws[1:]...)
}

func (h *WebHook) start(ctx context.Context) error {
	log.Trace("starting web service...")
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (h *WebHook) stop(ctx context.Context) error {
	log.Trace("stopping web service...")
	err := h.server.Shutdown(ctx)
	log.Trace("web service stopped")
	return err
}

func NewWebHook() *WebHook {
	return &WebHook{
		errChan: make(chan error),
	}
}

func loggerInjectorMiddleware(ctx context.Context) api.Middleware {
	l := log.FromContext(ctx)
	return api.MiddlewareFunc(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rCtx := r.Context()
			injectedReq := r.WithContext(log.InjectContext(rCtx, l))
			h.ServeHTTP(w, injectedReq)
		})
	})
}
