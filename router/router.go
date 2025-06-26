package router

import (
	"net/http"
	"reflect"
	"strings"

	"flugo.com/container"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)
type MiddlewareFunc func(HandlerFunc) HandlerFunc

type Route struct {
	Method      string
	Path        string
	Handler     HandlerFunc
	Middlewares []MiddlewareFunc
}

type Router struct {
	routes            []Route
	globalMiddlewares []MiddlewareFunc
	container         *container.Container
}

func NewRouter(c *container.Container) *Router {
	return &Router{
		routes:            make([]Route, 0),
		globalMiddlewares: make([]MiddlewareFunc, 0),
		container:         c,
	}
}

func (r *Router) Use(middleware MiddlewareFunc) {
	r.globalMiddlewares = append(r.globalMiddlewares, middleware)
}

func (r *Router) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.addRoute("GET", path, handler, middlewares)
}

func (r *Router) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.addRoute("POST", path, handler, middlewares)
}

func (r *Router) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.addRoute("PUT", path, handler, middlewares)
}

func (r *Router) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.addRoute("DELETE", path, handler, middlewares)
}

func (r *Router) addRoute(method, path string, handler HandlerFunc, middlewares []MiddlewareFunc) {
	route := Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Middlewares: middlewares,
	}
	r.routes = append(r.routes, route)
}

func (r *Router) RegisterController(controller interface{}, basePath string) {
	controllerType := reflect.TypeOf(controller)
	controllerValue := reflect.ValueOf(controller)

	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
		controllerValue = controllerValue.Elem()
	}

	r.container.Register(controller)

	for i := 0; i < controllerType.NumMethod(); i++ {
		method := controllerType.Method(i)
		methodValue := controllerValue.Method(i)

		if method.Type.NumIn() == 3 &&
			method.Type.In(1).Implements(reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()) &&
			method.Type.In(2) == reflect.TypeOf((*http.Request)(nil)) {

			httpMethod := extractHTTPMethod(method.Name)
			if httpMethod != "" {
				path := basePath + extractPath(method.Name)

				methodFunc := methodValue
				handler := func(w http.ResponseWriter, req *http.Request) {
					methodFunc.Call([]reflect.Value{
						reflect.ValueOf(w),
						reflect.ValueOf(req),
					})
				}
				r.addRoute(httpMethod, path, handler, nil)
			}
		}
	}
}

func extractHTTPMethod(methodName string) string {
	if strings.HasPrefix(methodName, "Get") {
		return "GET"
	}
	if strings.HasPrefix(methodName, "Post") {
		return "POST"
	}
	if strings.HasPrefix(methodName, "Put") {
		return "PUT"
	}
	if strings.HasPrefix(methodName, "Delete") {
		return "DELETE"
	}
	return ""
}

func extractPath(methodName string) string {
	for _, prefix := range []string{"Get", "Post", "Put", "Delete"} {
		if strings.HasPrefix(methodName, prefix) {
			remaining := methodName[len(prefix):]
			if remaining == "" {
				return ""
			}
			if strings.HasSuffix(remaining, "ById") {
				remaining = remaining[:len(remaining)-4] // Remove "ById"
				if remaining == "" {
					return "/{id}"
				}
				return "/" + strings.ToLower(remaining) + "/{id}"
			}
			return "/" + strings.ToLower(remaining)
		}
	}
	return "/" + strings.ToLower(methodName)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		if route.Method == req.Method && r.matchPath(route.Path, req.URL.Path) {
			handler := route.Handler

			for i := len(r.globalMiddlewares) - 1; i >= 0; i-- {
				handler = r.globalMiddlewares[i](handler)
			}

			for i := len(route.Middlewares) - 1; i >= 0; i-- {
				handler = route.Middlewares[i](handler)
			}

			handler(w, req)
			return
		}
	}

	http.NotFound(w, req)
}

func (r *Router) matchPath(routePath, requestPath string) bool {
	if routePath == requestPath {
		return true
	}

	if strings.HasPrefix(requestPath, routePath) &&
		(strings.HasSuffix(routePath, "/") ||
			len(requestPath) > len(routePath) && requestPath[len(routePath)] == '/') {
		return true
	}

	return false
}
