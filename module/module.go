package module

import (
	"flugo.com/container"
	"flugo.com/router"
)

type ModuleConfig struct {
	Controllers []ControllerConfig
	Providers   []interface{}
	Imports     []*Module
}

type ControllerConfig struct {
	Controller interface{}
	Path       string
}

type Module struct {
	config    ModuleConfig
	container *container.Container
	router    *router.Router
}

func NewModule(config ModuleConfig) *Module {
	return &Module{
		config: config,
	}
}

func (m *Module) Bootstrap(c *container.Container, r *router.Router) {
	m.container = c
	m.router = r

	for _, importedModule := range m.config.Imports {
		importedModule.Bootstrap(c, r)
	}

	for _, provider := range m.config.Providers {
		c.Register(provider)
	}

	for _, controllerConfig := range m.config.Controllers {
		r.RegisterController(controllerConfig.Controller, controllerConfig.Path)
	}
}
