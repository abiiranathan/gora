/*
Gora is a package for easily defining and matching routes using regular expressions.
Inspired by the Django web framework's powerful routing system,
this library allows you to define a set of patterns for your routes and
automatically match incoming requests to the appropriate handler.
Whether you are building a simple API or a complex web application,
this library is a convenient and flexible tool for routing requests in Go.
*/
package gora

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Enable Strict Trailing slash per URL
var StrictSlash bool = false

// Validation tag, default "validate"
var ValidationTag string = "validate"

// Debug /or production, defaults to false.
// If in production, turns off ConsoleWriter and writes to the io.Writer provide to the router.
var ModeProduction bool

// Panic with text if statement is false
func assert(statement bool, text string) {
	if !statement {
		panic(text)
	}
}

// Base Router implements the http.Handler interface.
type Router struct {
	routes     []route          // Stores all registered routes
	middleware []MiddlewareFunc // Stores all global middleware

	// Called if no path matches the request path.
	// Useful for handling SPA frontend applications
	notFound HandlerFunc

	// Request logger
	Logger *zerolog.Logger
}

// A single route. Stores url patterns, method and their corresponding handlers and middleware.
type route struct {
	pattern    *regexp.Regexp
	handler    func(ctx *Context)
	method     string
	middleware []MiddlewareFunc
}

func (r route) String() string {
	return fmt.Sprintf("/%s - %s", r.method, r.pattern)
}

type HandlerFunc func(ctx *Context)
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

/*
Configuration struct for embedding an embedded build directory.

In SPAMode, a catch-all route hooked up to send every failed request back to
index.html so that the client-side router can handle it.
*/
type StaticEmbed struct {
	EmbedFS        *embed.FS // The embed build/dist directory
	Route          string    //Route to serve your frontend application from. Default: "/"
	Dirname        string    // directory prefix for your build directory. Default: "build"
	IgnorePatterns []string  // path patterns to ignore. e.g "/api", "/ws"
	IndexFile      string    // Path to index.html relative, defaults to "index.html"
}

// Returns a pointer to a new router with the logging and recovery middleware applied.
// If you don't want whese middleware, call New() instead.
func Default(logOutput io.Writer) *Router {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logOutput})
	if ModeProduction {
		log.Logger = log.Output(zerolog.Logger{}.Output(logOutput))
	}

	r := &Router{Logger: &log.Logger}
	r.Use(RecoveryMiddleware, LoggerMiddleware)
	return r
}

// Returns a pointer to a new router.
// If you want logging middleware and recovery middleware applied,
// use Default() instead.
func New(logOutput io.Writer) *Router {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logOutput})
	if ModeProduction {
		log.Logger = log.Output(zerolog.Logger{}.Output(logOutput))
	}

	return &Router{Logger: &log.Logger}
}

// Apply middleware to the router.
func (r *Router) Use(middleware ...MiddlewareFunc) {
	assert(len(middleware) > 0, "len(middleware) must be greater than 0")
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) addRoute(pattern string, method string, handlers ...HandlerFunc) {
	assert(len(handlers) > 0, "You must pass in at least one handler function")

	r.routes = append(r.routes, route{
		compileRegex(pattern),
		handlers[len(handlers)-1],
		method,
		createMiddleware(handlers...)[:len(handlers)-1]})
}

// Create a new router group.
func (r *Router) Group(prefix string, middleware ...HandlerFunc) *RouterGroup {
	return &RouterGroup{r, prefix, createMiddleware(middleware...)}
}

// Serves the http request. Implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Initialize the context
	ctx := &Context{
		Request:   req,
		Response:  w,
		Params:    make(map[string]string),
		validator: NewValidator(ValidationTag),
		data:      make(map[string]any),
		Logger:    r.Logger,
		mu:        sync.RWMutex{},
	}

	// Loop through all routes until we find a match
	for _, route := range r.routes {
		if req.Method != route.method {
			continue
		}

		// Extract path parameters if the route pattern contains placeholders (e.g. /users/:id)
		path := req.URL.Path
		if StrictSlash && path[len(path)-1] != '/' {
			path += "/"
		}

		// Match route based on request path
		if route.pattern.MatchString(path) {
			matches := route.pattern.FindStringSubmatch(path)
			params := make(map[string]string)
			for i, name := range route.pattern.SubexpNames() {
				if i > 0 && i <= len(matches) {
					params[name] = matches[i]
				}
			}

			// Add the path parameters to the request context
			ctx.Params = params

			// Initialize a handler
			handler := route.handler

			// CombineMiddleare
			middleware := append(r.middleware, route.middleware...)
			for _, mw := range middleware {
				handler = mw(handler)
			}
			handler(ctx)
			return
		}
	}

	if r.notFound != nil {
		r.notFound(ctx)
		return
	}

	// If no matching route is found, return a 404 Not Found response
	http.NotFound(w, req)
}

func (r *Router) GET(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodGet, handlers...)
}

func (r *Router) POST(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodPost, handlers...)
}

func (r *Router) PUT(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodPut, handlers...)
}

func (r *Router) PATCH(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodPatch, handlers...)
}

func (r *Router) DELETE(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodDelete, handlers...)
}

func (r *Router) OPTIONS(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodOptions, handlers...)
}

func (r *Router) CONNECT(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodConnect, handlers...)
}

func (r *Router) TRACE(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodTrace, handlers...)
}

func (r *Router) HEAD(pattern string, handlers ...HandlerFunc) {
	r.addRoute(pattern, http.MethodHead, handlers...)
}

// Connect a handler to be called if no pattern matches the request path.
func (r *Router) NotFound(handler HandlerFunc) {
	// Global router middleware
	for _, mw := range r.middleware {
		handler = mw(handler)
	}
	r.notFound = handler
}

// Serve static files with the http.FileServer
// If root is not "/", you may want to provide a stripPrefix string
// to tidy up the request path. e.g
//
//	r := gora.Default()
//	r.Static("/static", ".", "/static")
func (r *Router) Static(root, dirname, stripPrefix string) {
	handler := http.StripPrefix(stripPrefix, http.FileServer(http.Dir(dirname)))
	handlerFunc := func(ctx *Context) {
		handler.ServeHTTP(ctx.Response, ctx.Request)
	}

	// Compile regex
	regex := regexp.MustCompile(root)
	r.routes = append(r.routes, route{regex, handlerFunc, http.MethodGet, nil})
}

// Serve files in an embedded directory.
// This is essential to embed build directories from frontend frameworks
// like svelte-kit, react, astro etc.
// Serves index.html at the root of the file system as if it was mounted at root.
// If ignore slice is not nil or empty, request path matching these routes are skipped.
func (r *Router) StaticEmbedFS(staticEmbed StaticEmbed) {
	fsys, err := fs.Sub(staticEmbed.EmbedFS, staticEmbed.Dirname)
	if err != nil {
		panic(err)
	}

	// Set default arguments
	if staticEmbed.IndexFile == "" {
		staticEmbed.IndexFile = "index.html"
	}

	if staticEmbed.Dirname == "" {
		staticEmbed.Dirname = "build"
	}

	if staticEmbed.Route == "" {
		staticEmbed.Route = "/"
	}

	// Initialize an http file system
	httpfs := http.FS(fsys)
	index, err := staticEmbed.EmbedFS.ReadFile(filepath.Join(staticEmbed.Dirname, staticEmbed.IndexFile))
	if err != nil {
		panic(err)
	}

	// Create a file server handler
	handler := http.FileServer(httpfs)

	// Helper to match request path to patterns to skip
	skipPath := func(path string) bool {
		skip := false
		for _, ignorePattern := range staticEmbed.IgnorePatterns {
			if strings.Contains(path, ignorePattern) {
				skip = true
			}
		}
		return skip
	}

	// Create a HandlerFunc
	handlerFunc := func(ctx *Context) {
		if skipPath(ctx.Request.URL.Path) {
			ctx.Abort(http.StatusNotFound, "Not Found")
			return
		}

		f, err := staticEmbed.EmbedFS.Open(filepath.Join(staticEmbed.Dirname, ctx.Request.URL.Path))
		if err != nil {
			if os.IsNotExist(err) {
				if filepath.Ext(ctx.Request.URL.Path) != "" {
					ctx.Status(http.StatusNotFound)
					return
				}

				// File not found. Let the client-side router handle the file
				ctx.Status(http.StatusAccepted)
				ctx.HTML(http.StatusOK, string(index))
			} else {
				// IO Error
				http.Error(ctx.Response, "something wrong happened!!", http.StatusInternalServerError)
			}
			return
		}

		// Close the file
		f.Close()

		// File exists let the fileServer handler deal with it
		handler.ServeHTTP(ctx.Response, ctx.Request)
	}

	r.routes = append(r.routes, route{compileRegex(staticEmbed.Route), handlerFunc, "GET", nil})

	// Catch-all route for SPA mode.
	r.NotFound(handlerFunc)

}

func (r *Router) Routes() []route {
	return r.routes
}

func pathPrefixToRegex(pathPrefix string) (string, error) {
	// Split the path prefix into its individual segments
	segments := strings.Split(pathPrefix, "/")

	// Initialize the regular expression
	regex := "^/"

	// Iterate through the segments
	numSigments := len(segments)
	for index, segment := range segments {
		// Check if the segment is a path parameter
		if strings.Contains(segment, "{") && strings.Contains(segment, "}") {
			// Extract the parameter name from the segment
			paramName := segment[1 : len(segment)-1]

			// Check if the parameter has a type specified
			if strings.Contains(paramName, ":") {
				// Extract the parameter name and type from the segment
				var paramType string
				params := strings.Split(paramName, ":")
				paramName = params[0]
				paramType = params[1]

				// Check the parameter type and add the corresponding regular expression to the regex
				if paramType == "int" {
					regex += "(?P<" + paramName + ">\\d+)"
				} else if paramType == "str" {
					regex += "(?P<" + paramName + ">\\w+)"
				} else if paramType == "float" {
					regex += "(?P<" + paramName + ">\\d+\\.\\d+)"
				} else if paramType == "bool" {
					regex += "(?P<" + paramName + ">true|false)"
				} else if paramType == "date" {
					regex += "(?P<" + paramName + ">\\d{4}-\\d{2}-\\d{2})"
				} else if paramType == "datetime" {
					regex += "(?P<" + paramName + ">\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2})"
				} else {
					return "", errors.New("invalid parameter type: " + paramType)
				}
			} else {
				// The parameter has no type specified, so consider it a string
				regex += "(?P<" + paramName + ">\\w+)"
			}
		} else {
			// Add the segment to the regex as is
			regex += segment
		}

		if index < numSigments-1 && segment != "" {
			regex += "/"
		}
	}

	// Add trailing slash if StrictSlash and path does not end in /
	if StrictSlash && len(regex) > 2 && regex[len(regex)-1] != '/' {
		regex += "/"
	}

	// Add the end anchor to the regex
	regex += "$"
	return regex, nil
}

func compileRegex(pat string) *regexp.Regexp {
	regex, err := pathPrefixToRegex(pat)
	if err != nil {
		panic(err)
	}
	return regexp.MustCompile(regex)
}

// Write data to a temporary file with given name.
// Returns absolute path to the file written to, function to delete the
// teporary directory where this file was created and an error if any.
// Temporary file created with permissions 0754.
func WriteToTempFile(name string, data []byte) (filename string, rmDir func(), err error) {
	dir, err := os.MkdirTemp("", "gora_temp")
	if err != nil {
		return "", nil, err
	}

	rmDir = func() { os.RemoveAll(dir) }
	filename = filepath.Join(dir, name)
	err = os.WriteFile(filename, data, 0754)
	if err != nil {
		return "", nil, err
	}
	return filename, rmDir, nil
}
