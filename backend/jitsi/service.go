package jitsi

import (
	"crypto/rsa"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"sidus.io/home-call/util"
	"time"
)

type App struct {
	appId    string
	appKeyId string
	appKey   *rsa.PrivateKey
}

func NewApp(
	appId string,
	appKeyId string,
	appKey *rsa.PrivateKey,
) *App {
	return &App{
		appId:    appId,
		appKeyId: appKeyId,
		appKey:   appKey,
	}
}

func (s *App) NewCall() (*Call, error) {
	roomName, err := util.RandomString(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random room name: %w", err)

	}

	return &Call{
		app:      s,
		roomName: roomName,
	}, nil
}

func (s *App) jitsiJWT(
	roomName,
	userName,
	userId string,
) (string, error) {
	claims := JitsiClaims{
		Room: roomName,
		Context: JitsiClaimContext{
			User: JitsiClaimUser{
				ID:                 userId,
				Name:               userName,
				Avatar:             "",
				Email:              "",
				Moderator:          false,
				HiddenFromRecorder: false,
			},
			Features: JitsiClaimFeatures{
				Livestreaming: false,
				OutboundCall:  false,
				Transcription: false,
				Recording:     false,
			},
		},
		RegisteredClaims: jwt.RegisteredClaims{
			// Jitsi requires audience to be set as a string
			//Audience:  []string{"jitsi"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chat",
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   s.appId,
		},
		Audience: "jitsi",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.appKeyId

	tokenString, err := token.SignedString(s.appKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	return tokenString, nil
}
