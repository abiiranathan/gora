package gora

import (
	"net/http"
	"strings"
	"time"

	"github.com/mileusna/useragent"
)

// Custom server recovery middleware.
func RecoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx.Logger.Info().Err(err.(error)).Msg("panic")
				http.Error(ctx.Response, "InternalServerError", http.StatusInternalServerError)
			}
		}()
		next(ctx)
	}
}

// A simple logging middleware.
// Logs the Request Method, Path, IP, Browser, Latency.
func LoggerMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		ua := useragent.Parse(ctx.Request.Header.Get("User-Agent"))
		// Get the IP address of the client
		var ip string = ctx.Request.RemoteAddr

		if strings.HasPrefix(ip, "[::1]") {
			ip = "localhost"
		} else {
			hostPortpair := strings.Split(ip, ":")
			if len(hostPortpair) == 2 {
				ip = hostPortpair[0]
			}
		}

		start := time.Now()
		next(ctx)
		latency := time.Since(start).String()
		ctx.Logger.Info().
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Str("ip", ip).
			Str("browser", ua.Name).
			Str("device", ua.Device).
			Str("os", ua.OS).
			Str("os-version", ua.OSVersion).
			Str("latency", latency).
			Msg("")
	}
}

// Wraps a standard http.HandlerFunc and returns HandlerFunc
// that this router expects.
func WrapHttpHandler(h http.Handler) HandlerFunc {
	return func(ctx *Context) {
		h.ServeHTTP(ctx.Response, ctx.Request)
	}
}

// Wraps a standard http.HandlerFunc and returns HandlerFunc
// that this router expects.
func WrapHttpHandlerFunc(h http.HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		h(ctx.Response, ctx.Request)
	}
}

func createMiddleware(handlers ...HandlerFunc) []MiddlewareFunc {
	funcs := make([]MiddlewareFunc, 0, len(handlers))
	for _, hf := range handlers {
		funcs = append(funcs, func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				hf(ctx)
			}
		})
	}

	return funcs
}
