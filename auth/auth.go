package auth

import (
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("jwt token is invalid")

// Hashes a password string using default cost
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Compares the password with the hash using bcrypt.CompareHashAndPassword
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type Tokener interface {
	Create(id uint) (string, error)
	Verify(token string) (uint, error)
}

/*
JWT implements the Tokener interface.
Create method takes in id as the payload.

It is important to note that bcrypt is a slow hashing function, which means that it can take a long time to generate and verify hashes. This is a security feature, as it makes it more difficult for attackers to guess or brute-force the hashes. However, it also means that bcrypt may not be the most suitable choice for generating and verifying tokens, especially if you need to do it frequently or in high-concurrency situations.
*/
type JWT struct {
	secretKey     string            // The SECRET_KEY used by bcrypt to create and verify tokens
	expireAfter   time.Duration     // Time to expire for the jwt, default: 72 hours
	signingMethod jwt.SigningMethod // Signing method, default: jwt.SigningMethodHS256
}

type JWTOption func(*JWT)

func SigningMethod(method jwt.SigningMethod) JWTOption {
	return func(j *JWT) {
		j.signingMethod = method
	}
}

func ExpiresAfter(expiresAfter time.Duration) JWTOption {
	return func(j *JWT) {
		j.expireAfter = expiresAfter
	}
}

func NewJWT(secretKey string, options ...JWTOption) Tokener {
	jwtoken := &JWT{
		signingMethod: jwt.SigningMethodHS256,
		expireAfter:   time.Hour * 72,
		secretKey:     secretKey,
	}

	for _, opt := range options {
		opt(jwtoken)
	}

	return jwtoken
}

// Creates a jwt token that expires after the configured duration.
//
// Payload is the id.
// Returns a base64 encoded JWT string.
func (jwtoken *JWT) Create(id uint) (string, error) {
	token := jwt.New(jwtoken.signingMethod)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = id
	claims["exp"] = time.Now().Add(jwtoken.expireAfter).Unix()
	encodedString, err := token.SignedString([]byte(jwtoken.secretKey))
	return base64.StdEncoding.EncodeToString([]byte(encodedString)), err
}

// Verifies a base64 encoded token string
// Returns the user id from the payload and an error if any or nil.
// If the base64Token can not be decoded or an error occurs in jwt.Parse,
// the error is auth.ErrInvalidToken
func (jwtoken *JWT) Verify(base64Token string) (uint, error) {
	tokenString, err := base64.StdEncoding.DecodeString(base64Token)
	if err != nil {
		return 0, ErrInvalidToken
	}

	token, err := jwt.Parse(string(tokenString), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(jwtoken.secretKey), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		id := uint(claims["id"].(float64))
		return id, nil
	}
	return 0, ErrInvalidToken
}
