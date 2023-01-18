package gora

import (
	"net/http"
	"strconv"
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

// CorsConfig is the configuration struct that sets up the cors middleware.
// All values are passed to the Headers as provided without special defaults.
type CorsConfig struct {

	/*
		The Access-Control-Allow-Origin response header indicates whether the response can be
		shared with requesting code from the given origin.

		Passing in []string{"*"} will allow all origins
	*/
	AllowedOrigins []string

	/*
		The Access-Control-Allow-Methods response header specifies one or more methods allowed
		when accessing a resource in response to a preflight request
	*/
	AllowedMethods []string

	/*
		The Access-Control-Allow-Headers response header is used in response to a preflight request
		which includes the Access-Control-Request-Headers to indicate which HTTP headers can be used
		 during the actual request
	*/
	AllowedHeaders []string

	/*
		The Access-Control-Expose-Headers response header allows a server to indicate which response headers should be made available to scripts running in the browser, in response to a cross-origin request
	*/
	ExposeHeaders []string

	/*
		The Access-Control-Allow-Credentials response header tells browsers whether to expose the response to the frontend JavaScript code when the request's credentials mode (Request.credentials) is include
	*/
	AllowCredentials bool

	/*
		The Access-Control-Max-Age response header indicates how long the results of a preflight request (that is the information contained in the Access-Control-Allow-Methods and Access-Control-Allow-Headers headers) can be cached.
	*/
	MaxAge time.Duration
}

func (m *CorsConfig) isOriginAllowed(origin string) bool {
	for _, o := range m.AllowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// Cors middleware.
// m CorsConfig configures the CORS response headers.
func Cors(m CorsConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			origin := c.Request.Header.Get("Origin")
			if origin == "" || !m.isOriginAllowed(origin) {
				c.Abort(http.StatusForbidden, "Forbidden")
				return
			}

			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(m.AllowedMethods, ","))
			c.Header("Access-Control-Allow-Headers", strings.Join(m.AllowedHeaders, ","))
			c.Header("Access-Control-Expose-Headers", strings.Join(m.ExposeHeaders, ","))
			c.Header("Access-Control-Max-Age", strconv.Itoa(int(m.MaxAge.Seconds())))

			if m.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			if c.Request.Method == http.MethodOptions {
				c.Status(http.StatusOK)
				return
			}

			next(c)
		}
	}
}
