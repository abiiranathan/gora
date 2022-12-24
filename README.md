# Gora

Gora is a routing library for Go that allows you to easily define and match routes using regular expressions. Inspired by the Django web framework's powerful routing system, this library allows you to define a set of patterns for your routes and automatically match incoming requests to the appropriate handler.

Whether you are building a simple API or a complex web application, Gora is a convenient and flexible tool for routing requests in Go.

## Features

- Define routes using regular expressions
- Automatic matching of incoming requests to the appropriate handler
- Support for middleware
- Path parameters
- Request context object with access to the incoming request, response writer, and various utilities
- Automatic request body binding from JSON and XML to struct and validation with [go validator](https://github.com/go-playground/validator)
- Support for route groups
- Structured logging with [zerolog](https://github.com/rs/zerolog)
- Standard library support through wrapper functions.

### Example

```go
package main

import (
	"embed"
	"net/http"

	"github.com/abiiranathan/gora/gora"
)

//go:embed all:build
var build embed.FS

func main() {
	r := gora.Default()

	r.GET("/api", func(ctx *gora.Context) {
		ctx.Text(http.StatusOK, "Hello World!")
	})

    r.GET("/users/{userId:int}", func(ctx *gora.Context) {
        userId, _ := ctx.IntParam("userId")
		ctx.JSON(http.StatusOK, userId)
	})

	r.StaticEmbedFS(
		gora.StaticEmbed{
			EmbedFS: &build,
			Route:   "/",
			Dirname: "build"},
	)

    // Start a server with graceful shutdown configured
	r.Run(":8080") // or RunTLS(...)
}
```

Documentation
Full documentation for Gora can be found at https://abiiranathan.github.io/gora/.

Installation
To install Gora, use go get:
`go get github.com/abiiranathan/gora`

Contribution
We welcome contributions to Gora! If you have an idea for a new feature or have found a bug, please open an issue on GitHub.

License
Gora is released under the MIT License. See the LICENSE file for more details.
