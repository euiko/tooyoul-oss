package app

import (
	"errors"
	"sync"

	"github.com/euiko/tooyoul/mineman/lib/app/api"
)

type ModuleFactory func() api.Module

type ModuleRegistry struct {
	registries sync.Map
}

func (r *ModuleRegistry) Unregister(name string) {
	r.registries.Delete(name)
}

func (r *ModuleRegistry) Register(name string, factory ModuleFactory) {
	r.registries.Store(name, factory)
}

func (r *ModuleRegistry) IsRegistered(name string) bool {
	_, ok := r.registries.Load(name)
	return ok
}

func (r *ModuleRegistry) Get(name string) (ModuleFactory, error) {
	value, ok := r.registries.Load(name)
	if !ok {
		return nil, errors.New("couldn't find appropriate module")
	}

	return value.(ModuleFactory), nil
}

func (r *ModuleRegistry) Load() []ModuleFactory {
	factories := []ModuleFactory{}

	r.registries.Range(func(key, value interface{}) bool {
		factories = append(factories, value.(ModuleFactory))
		return false
	})

	return factories
}
