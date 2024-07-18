package auth

import (
	"connectrpc.com/connect"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/coreos/go-oidc/v3/oidc"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"golang.org/x/oauth2"
	"gopkg.in/go-jose/go-jose.v2/jwt"
	"net/http"
	"net/url"
	"sidus.io/home-call/gen/jetdb/public/model"
	. "sidus.io/home-call/gen/jetdb/public/table"
	"strings"
	"time"
)

type AuthInterceptor struct {
	jwtValidator *validator.Validator
	oidcProvider *oidc.Provider
	noVerify     bool
	db           *sql.DB
}

func NewAuthInterceptor(issuer *url.URL, audience string, noVerify bool, db *sql.DB) (*AuthInterceptor, error) {
	var jwtValidator *validator.Validator
	var oidcProvider *oidc.Provider
	if !noVerify {
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

		oidcProvider, err = oidc.NewProvider(context.Background(), issuer.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create oidc provider: %w", err)
		}
	}

	return &AuthInterceptor{
		jwtValidator: jwtValidator,
		noVerify:     noVerify,
		oidcProvider: oidcProvider,
		db:           db,
	}, nil
}

func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		ctx, err := i.authenticate(ctx, req.Header())
		if err != nil {
			return nil, fmt.Errorf("failed to verify token: %w", err)
		}
		return next(ctx, req)
	}
}

func (*AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		ctx, err := i.authenticate(ctx, conn.RequestHeader())
		if err != nil {
			return fmt.Errorf("failed to verify token: %w", err)
		}

		return next(ctx, conn)
	}
}

func (i *AuthInterceptor) authenticate(ctx context.Context, header http.Header) (context.Context, error) {
	token := strings.TrimSpace(
		strings.TrimPrefix(
			header.Get("Authorization"),
			"Bearer ",
		),
	)

	if token == "" {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing token"))
	}

	var claims *validator.ValidatedClaims
	if i.noVerify {
		parsedToken, err := jwt.ParseSigned(token)
		if err != nil {
			return ctx, fmt.Errorf("could not parse the token: %w", err)
		}

		jwtClaims := jwt.Claims{}
		err = parsedToken.UnsafeClaimsWithoutVerification(&jwtClaims)
		if err != nil {
			return ctx, fmt.Errorf("could not parse the claims: %w", err)
		}
		claims = &validator.ValidatedClaims{
			RegisteredClaims: validator.RegisteredClaims{
				Issuer:    jwtClaims.Issuer,
				Subject:   jwtClaims.Subject,
				Audience:  jwtClaims.Audience,
				Expiry:    jwtClaims.Expiry.Time().Unix(),
				NotBefore: jwtClaims.NotBefore.Time().Unix(),
				IssuedAt:  jwtClaims.IssuedAt.Time().Unix(),
				ID:        jwtClaims.ID,
			},
		}
	} else {
		parsedToken, err := i.jwtValidator.ValidateToken(ctx, token)
		if err != nil {
			return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("failed to validate token: %w", err))
		}

		var ok bool
		claims, ok = parsedToken.(*validator.ValidatedClaims)
		if !ok {
			return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unexpected claims type"))
		}
	}
	userInfo, err := i.getUserInfo(ctx, token, claims.RegisteredClaims.Subject)
	if err != nil {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("failed to get user info: %w", err))
	}

	return WithAuth(ctx, &Auth{
		Subject:       claims.RegisteredClaims.Subject,
		DisplayName:   userInfo.Name,
		VerifiedEmail: userInfo.Email,
	}), nil
}

func (i *AuthInterceptor) getUserInfo(ctx context.Context, token string, subject string) (parsedUserInfo, error) {
	// Get the user info from cache
	userInfo, err := i.getUserInfoFromCache(ctx, token)
	if err == nil {
		return userInfo, nil
	}
	if !errors.Is(err, ErrCacheMiss) {
		return parsedUserInfo{}, fmt.Errorf("failed to get user info from cache: %w", err)
	}

	if i.noVerify {
		userInfo.Subject = subject
		userInfo.EmailVerified = true
		userInfo.Email = normalizeEmail(strings.TrimPrefix(subject, "user:"))
		userInfo.Name = strings.ToTitle(strings.Split(strings.TrimPrefix(subject, "user:"), "@")[0])
	} else {
		rawUserInfo, err := i.oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
		if err != nil {
			return parsedUserInfo{}, fmt.Errorf("failed to get user info: %w", err)
		}

		err = rawUserInfo.Claims(&userInfo)
		if err != nil {
			return parsedUserInfo{}, fmt.Errorf("failed to parse user info: %w", err)
		}
	}

	if userInfo.Subject != subject {
		return parsedUserInfo{}, fmt.Errorf("subject mismatch")
	}

	userInfo.Email = normalizeEmail(userInfo.Email)
	if !userInfo.EmailVerified || userInfo.Email == "" {
		return parsedUserInfo{}, fmt.Errorf("email not verified")
	}

	// Cache the user info
	err = i.storeUserInfoInCache(ctx, token, userInfo)
	if err != nil {
		return parsedUserInfo{}, fmt.Errorf("failed to store user info: %w", err)

	}

	// Store the user info in the database (only on cache miss)
	err = i.ensureUserData(ctx, userInfo)
	if err != nil {
		return parsedUserInfo{}, fmt.Errorf("failed to store user data: %w", err)
	}

	return userInfo, nil
}

type parsedUserInfo struct {
	Subject       string `json:"sub"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (i *AuthInterceptor) ensureUserData(ctx context.Context, userInfo parsedUserInfo) error {
	userModel := model.User{
		Email:       userInfo.Email,
		DisplayName: userInfo.Name,
		IdpUserID:   userInfo.Subject,
	}
	stmt := User.
		INSERT(User.IdpUserID, User.Email, User.DisplayName).
		MODEL(userModel).
		ON_CONFLICT(User.IdpUserID).
		DO_UPDATE(SET(
			User.Email.SET(String(userModel.Email)),
			User.DisplayName.SET(String(userModel.DisplayName)),
		))

	_, err := stmt.ExecContext(ctx, i.db)
	if err != nil {
		return fmt.Errorf("failed to ensure user info: %w", err)
	}
	return nil
}

func (i *AuthInterceptor) getUserInfoFromCache(ctx context.Context, token string) (parsedUserInfo, error) {
	var dest model.UserinfoCache
	err := SELECT(UserinfoCache.Userinfo).
		FROM(UserinfoCache).
		WHERE(UserinfoCache.TokenHash.EQ(String(hashToken(token)))).
		QueryContext(ctx, i.db, &dest)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return parsedUserInfo{}, ErrCacheMiss
		}
		return parsedUserInfo{}, fmt.Errorf("failed to query database: %w", err)
	}

	var userInfo parsedUserInfo
	err = json.Unmarshal([]byte(dest.Userinfo), &userInfo)
	if err != nil {
		return parsedUserInfo{}, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return userInfo, nil
}

var ErrCacheMiss = errors.New("cache miss")

func (i *AuthInterceptor) storeUserInfoInCache(ctx context.Context, token string, userInfo parsedUserInfo) error {
	userInfoBytes, err := json.Marshal(userInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	stmt := UserinfoCache.
		INSERT(UserinfoCache.TokenHash, UserinfoCache.Userinfo).
		VALUES(hashToken(token), string(userInfoBytes)).
		ON_CONFLICT(UserinfoCache.TokenHash).
		DO_UPDATE(SET(
			UserinfoCache.Userinfo.SET(String(string(userInfoBytes))),
		))

	_, err = stmt.ExecContext(ctx, i.db)
	if err != nil {
		return fmt.Errorf("failed to store user info: %w", err)
	}

	return nil
}

func hashToken(token string) string {
	hasher := sha1.New()
	hasher.Write([]byte(token))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
