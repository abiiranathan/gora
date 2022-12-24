package gora

import "net/http"

// Router group allows for nesting of url routes.
// Has similar methods as the base router.
type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

func (g *RouterGroup) addRoute(pattern string, method string, handlers ...HandlerFunc) {
	assert(len(handlers) > 0, "You must pass in at least one handler function")

	g.router.routes = append(g.router.routes, route{
		compileRegex(pattern),
		handlers[len(handlers)-1],
		method,
		createMiddleware(handlers...)[:len(handlers)-1]})
}

func (g *RouterGroup) Use(middleware ...HandlerFunc) {
	g.middleware = append(g.middleware, createMiddleware(middleware...)...)
}

func (g *RouterGroup) GET(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodGet, handlers...)
}

func (g *RouterGroup) POST(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPost, handlers...)
}

func (g *RouterGroup) PUT(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPut, handlers...)
}

func (g *RouterGroup) PATCH(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodPatch, handlers...)
}

func (g *RouterGroup) DELETE(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodDelete, handlers...)
}

// Other methods
func (g *RouterGroup) OPTIONS(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodOptions, handlers...)
}

func (g *RouterGroup) CONNECT(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodConnect, handlers...)
}

func (g *RouterGroup) TRACE(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodTrace, handlers...)
}

func (g *RouterGroup) HEAD(pattern string, handlers ...HandlerFunc) {
	g.addRoute(g.prefix+pattern, http.MethodHead, handlers...)
}
