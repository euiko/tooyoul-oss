package event

import (
	"github.com/euiko/tooyoul/mineman/pkg/app"
)

var moduleRegistry app.ModuleRegistry

func RegisterBroker(name string, factory app.ModuleFactory) {
	moduleRegistry.Register(name, factory)
}

func GetBroker(name string) (app.ModuleFactory, error) {
	return moduleRegistry.Get(name)
}

func LoadBrokers() []app.ModuleFactory {
	return moduleRegistry.Load()
}

func LoadBrokersMap() map[string]app.ModuleFactory {
	return moduleRegistry.LoadMap()
}
