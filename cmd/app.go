package cmd

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"flugo.com/auth"
	"flugo.com/cache"
	"flugo.com/config"
	"flugo.com/container"
	"flugo.com/logger"
	"flugo.com/middleware"
	"flugo.com/module"
	"flugo.com/router"
	"flugo.com/upload"
)

type Application struct {
	container *container.Container
	router    *router.Router
	modules   []*module.Module
	config    *config.Config
}

func (a *Application) Start() {
	panic("unimplemented")
}

func NewApplication() *Application {
	cfg := config.Load()

	logger.Init(&cfg.Logger)
	cache.Init(1000, 30*time.Minute)
	auth.Init(&cfg.JWT)
	upload.Init(&cfg.Upload)

	c := container.NewContainer()
	r := router.NewRouter(c)

	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	return &Application{
		container: c,
		router:    r,
		modules:   make([]*module.Module, 0),
		config:    cfg,
	}
}

func (a *Application) RegisterModule(m *module.Module) {
	a.modules = append(a.modules, m)
	m.Bootstrap(a.container, a.router)
}

func (a *Application) Use(middleware router.MiddlewareFunc) {
	a.router.Use(middleware)
}

func (a *Application) GET(path string, handler router.HandlerFunc, middlewares ...router.MiddlewareFunc) {
	a.router.GET(path, handler, middlewares...)
}

func (a *Application) POST(path string, handler router.HandlerFunc, middlewares ...router.MiddlewareFunc) {
	a.router.POST(path, handler, middlewares...)
}

func (a *Application) PUT(path string, handler router.HandlerFunc, middlewares ...router.MiddlewareFunc) {
	a.router.PUT(path, handler, middlewares...)
}

func (a *Application) DELETE(path string, handler router.HandlerFunc, middlewares ...router.MiddlewareFunc) {
	a.router.DELETE(path, handler, middlewares...)
}

func (a *Application) Listen(port int) error {
	address := fmt.Sprintf(":%d", port)
	log.Printf("Server starting on port %d", port)
	return http.ListenAndServe(address, a.router)
}

func Bootstrap(modules ...*module.Module) *Application {
	app := NewApplication()

	for _, m := range modules {
		app.RegisterModule(m)
	}

	return app
}
