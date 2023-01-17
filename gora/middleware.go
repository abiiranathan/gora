package gora

import (
	"net/http"
	"strings"
	"time"

	"github.com/mileusna/useragent"
	"github.com/rs/zerolog"
)

// Custom server recovery middleware.
func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				switch val := err.(type) {
				case string:
					ctx.Logger.Info().Str("message", val).Msg("internal server error")
					ctx.Status(http.StatusInternalServerError).HTML(val)
				case error:
					ctx.Logger.Info().Str("message", val.Error()).Msg("internal server error")
					ctx.Status(http.StatusInternalServerError).HTML(val.Error())
				default:
					ctx.Logger.Info().Interface("message", val).Msg("internal server error")
					ctx.Status(http.StatusInternalServerError).HTML("Something went wrong!")
				}
			}
		}()

		next(ctx)
	}
}

// A simple logging middleware.
// Logs the Request Method, Path, IP, Browser, Latency.
func Logger(next HandlerFunc) HandlerFunc {
	zerolog.TimeFieldFormat = time.RFC3339

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
			Int("statusCode", ctx.StatusCode()).
			Str("ip", ip).
			Str("browser", ua.Name).
			Str("device", ua.Device).
			Str("os", ua.OS).
			Str("os-version", ua.OSVersion).
			Str("latency", latency).
			Msg("")
	}
}

// Wraps a standard http.Handler.
func WrapH(h http.Handler) HandlerFunc {
	return func(ctx *Context) {
		h.ServeHTTP(ctx.Response, ctx.Request)
	}
}

// Wraps a standard http.HandlerFunc.
func WrapHF(h http.HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		h(ctx.Response, ctx.Request)
	}
}
