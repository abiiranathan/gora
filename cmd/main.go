package main

import (
	"embed"
	"net/http"
	"os"

	"github.com/abiiranathan/gora/gora"
)

//go:embed all:build
var build embed.FS

type user struct {
	ID int `json:"id"`
}

func main() {
	gora.ModeProduction = false
	r := gora.Default(os.Stdout)

	r.Use(func(next gora.HandlerFunc) gora.HandlerFunc {
		return func(ctx *gora.Context) {
			ctx.Set("user", user{10})
			next(ctx)
		}
	})

	r.GET("/api", func(ctx *gora.Context) {
		u := ctx.MustGet("user")
		ctx.JSON(http.StatusOK, u)
	})

	r.Static("/", ".", "")
	r.StaticEmbedFS(gora.StaticEmbed{EmbedFS: &build, Route: "/", Dirname: "build"})
	r.Run(":8080")
}
