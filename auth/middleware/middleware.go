package middleware

import (
	"fmt"
	"net/http"

	"github.com/abiiranathan/gora/auth"
	"github.com/abiiranathan/gora/gora"
)

// UserLoader function loads user from the database given the id.
// Returns the user and an error if user can not be loaded or user is not active.
type UserLoader func(userId uint) (user any, err error)

/*
LoginRequired when called with secretKey and userLoader creates a jwt
middleware that automatically extracts jwt from the request header,
verifies it fetches the user using the userLoader function and attaches it to the context
with the key "user"
Usage:

	secretKey := os.Getenv("SECRET_KEY")
	func userLoader(id uint) (models.User, error){
		return repo.FetchUser(id)
	}
	AuthMiddleware := LoginRequired(secretKey, userLoader)

	r := gora.Default()
	r.Use(AuthMiddleware)
*/
func LoginRequired(secretKey string, userLoader UserLoader) gora.MiddlewareFunc {
	tokener := auth.NewJWT(secretKey)

	return func(next gora.HandlerFunc) gora.HandlerFunc {
		return func(ctx *gora.Context) {
			// Get the Bearer token from the request
			token := ctx.BearerToken()
			if token == "" {
				ctx.Abort(http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Verify the token
			userId, err := tokener.Verify(token)
			if err != nil {
				ctx.Abort(http.StatusUnauthorized, fmt.Sprintf("Unauthorized: %s", err.Error()))
				return
			}

			// Fetch user by id
			user, err := userLoader(userId)
			if err != nil {
				ctx.Abort(http.StatusForbidden, "Forbidden: User not found!")
				return
			}

			// Set user in the context
			ctx.Set("user", user)
			next(ctx)
		}
	}
}
