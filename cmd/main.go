package main

import (
	"embed"
	"net/http"

	"github.com/abiiranathan/gora/auth"
	"github.com/abiiranathan/gora/auth/middleware"
	"github.com/abiiranathan/gora/gora"
	"github.com/abiiranathan/gora/ws"
)

//go:embed all:build
var build embed.FS

type User struct {
	ID int `json:"id"`
}

func main() {
	gora.ModeProduction = false
	r := gora.Default()

	LoginMiddleware := middleware.LoginRequired("secretKey", func(userId uint) (user User, err error) {
		return User{ID: 10}, nil
	})

	tokener := auth.NewJWT("secretKey")

	api := r.Group("/api")
	{
		api.POST("/register", func(c *gora.Context) {
			var u User

			err := c.MustBindJSON(&u)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}

			t, err2 := tokener.Create(uint(u.ID))
			if err2 != nil {
				c.AbortWithError(http.StatusInternalServerError, err2)
				return
			}

			c.JSON(gora.Map{"token": t})
		})

		api.GET("/user", func(c *gora.Context) {
			user := c.MustGet("user").(User)
			c.JSON(user)
		}, LoginMiddleware)
	}

	hub, quit := ws.NewHandler()
	defer quit()

	go hub.Run()
	r.GET("/ws", gora.WrapH(hub))

	// r.Static("/", ".", "")
	r.StaticEmbedFS(gora.StaticEmbed{
		EmbedFS:        &build,
		Route:          "/",
		Dirname:        "build",
		IgnorePatterns: []string{"/api", "/ws"}})
	r.Run(":8080")
}
