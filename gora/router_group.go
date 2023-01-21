package gora

import (
	"net/http"
)

// Router group allows for nesting of url routes.
// Has similar methods as the base router.
type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

func (g *RouterGroup) addRoute(pattern string, method string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.router.routes = append(g.router.routes, route{
		pattern:    compileRegex(pattern),
		handler:    handler,
		method:     method,
		middleware: middleware})
}

func (g *RouterGroup) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *RouterGroup) GET(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodGet, handler, middleware...)
}

func (g *RouterGroup) POST(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPost, handler, middleware...)
}

func (g *RouterGroup) PUT(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPut, handler, middleware...)
}

func (g *RouterGroup) PATCH(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPatch, handler, middleware...)
}

func (g *RouterGroup) DELETE(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodDelete, handler, middleware...)
}

// Other methods
func (g *RouterGroup) OPTIONS(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodOptions, handler, middleware...)
}

func (g *RouterGroup) CONNECT(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodConnect, handler, middleware...)
}

func (g *RouterGroup) TRACE(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodTrace, handler, middleware...)
}

func (g *RouterGroup) HEAD(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	g.addRoute(g.prefix+pattern, http.MethodHead, handler, middleware...)
}

func (g *RouterGroup) Static(pattern, dirname, stripPrefix string) {
	handler := http.StripPrefix(stripPrefix, http.FileServer(http.Dir(dirname)))
	handlerFunc := func(ctx *Context) {
		ctx.Response.WriteHeader(http.StatusOK)
		handler.ServeHTTP(ctx.Response, ctx.Request)
	}
	g.addRoute(g.prefix+pattern, http.MethodGet, handlerFunc, nil)
}
