package api

import "net/http"

type (
	Middleware interface {
		Handle(h http.Handler) http.Handler
	}
	MiddlewareFunc func(h http.Handler) http.Handler

	WebService interface {
		CreateEndpoints(middlewares ...Middleware) []Endpoint
	}

	SkipDefaultMiddlewaresExt interface {
		SkipDefaultMiddlewares() bool
	}

	SkipMiddlewaresExt interface {
		SkipMiddlewares() bool
	}

	Endpoint struct {
		Middlewares []Middleware
		Path        string
		Method      string
		Handler     http.Handler
	}
)

func (m MiddlewareFunc) Handle(h http.Handler) http.Handler {
	return m(h)
}
