package jitsi

import "github.com/golang-jwt/jwt/v5"

type JitsiClaims struct {
	Room    string            `json:"room"`
	Context JitsiClaimContext `json:"context"`
	jwt.RegisteredClaims
	Audience string `json:"aud"`
}

type JitsiClaimContext struct {
	User     JitsiClaimUser     `json:"user"`
	Features JitsiClaimFeatures `json:"features"`
}

type JitsiClaimUser struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Avatar             string `json:"avatar"`
	Email              string `json:"email"`
	Moderator          bool   `json:"moderator"`
	HiddenFromRecorder bool   `json:"hidden-from-recorder"`
}

type JitsiClaimFeatures struct {
	Livestreaming bool `json:"livestreaming"`
	OutboundCall  bool `json:"outbound-call"`
	Transcription bool `json:"transcription"`
	Recording     bool `json:"recording"`
}
