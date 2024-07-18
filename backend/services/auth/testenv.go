package auth

import (
	"connectrpc.com/connect"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

func WithDummyToken[T any](subject string, request *connect.Request[T]) *connect.Request[T] {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": fmt.Sprintf("user:%s", subject),
	})
	tokenString, err := jwtToken.SignedString([]byte("secret"))
	if err != nil {
		panic(err)
	}

	request.Header().Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
	return request
}
