package auth

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"gopkg.in/go-jose/go-jose.v2/jwt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type authInterceptor struct {
	jwtValidator *validator.Validator
	noVerify     bool
}

func NewAuthInterceptor(issuer *url.URL, audience string, noVerify bool) (*authInterceptor, error) {

	var jwtValidator *validator.Validator
	if noVerify {
		jwtValidator = nil
	} else {
		provider := jwks.NewCachingProvider(issuer, 5*time.Minute)
		var err error
		jwtValidator, err = validator.New(
			provider.KeyFunc,
			validator.RS256,
			issuer.String(),
			[]string{audience},
			validator.WithAllowedClockSkew(time.Minute*5),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create jwt validator: %w", err)
		}
	}

	return &authInterceptor{
		jwtValidator: jwtValidator,
		noVerify:     noVerify,
	}, nil
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		ctx, err := i.verifyToken(ctx, req.Header())
		if err != nil {
			return nil, fmt.Errorf("failed to verify token: %w", err)
		}
		return next(ctx, req)
	}
}

func (*authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		ctx, err := i.verifyToken(ctx, conn.RequestHeader())
		if err != nil {
			return fmt.Errorf("failed to verify token: %w", err)
		}

		return next(ctx, conn)
	}
}

func (i *authInterceptor) verifyToken(ctx context.Context, header http.Header) (context.Context, error) {
	token := strings.TrimSpace(
		strings.TrimPrefix(
			header.Get("Authorization"),
			"Bearer ",
		),
	)

	if token == "" {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing token"))
	}

	if i.noVerify {
		parsedToken, err := jwt.ParseSigned(token)
		if err != nil {
			return ctx, fmt.Errorf("could not parse the token: %w", err)
		}

		claims := jwt.Claims{}
		err = parsedToken.UnsafeClaimsWithoutVerification(&claims)
		if err != nil {
			return ctx, fmt.Errorf("could not parse the claims: %w", err)
		}

		return WithAuth(ctx, &Auth{
			Subject: claims.Subject,
		}), nil
	}

	parsedToken, err := i.jwtValidator.ValidateToken(ctx, token)
	if err != nil {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("failed to validate token: %w", err))
	}

	claims, ok := parsedToken.(*validator.ValidatedClaims)
	if !ok {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unexpected claims type"))
	}

	return WithAuth(ctx, &Auth{
		Subject: claims.RegisteredClaims.Subject,
	}), nil
}
