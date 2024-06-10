package auth

import "context"

type key int

var authKey key = 0

type Auth struct {
	Subject string
}

func WithAuth(ctx context.Context, auth *Auth) context.Context {
	return context.WithValue(ctx, authKey, auth)
}

func GetAuth(ctx context.Context) *Auth {
	auth, _ := ctx.Value(authKey).(*Auth)
	return auth
}
