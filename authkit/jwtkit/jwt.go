package jwtkit

import (
	"errors"
	"fmt"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/plainq/servekit/errkit"
)

// TokenManager is an interface that holds the logic of token management.
type TokenManager interface {
	// Sign takes a Token and signs it.
	Sign(token *Token) (string, error)

	// Verify takes a token string and verifies it.
	Verify(token string) error

	// ParseVerify takes a token string and parses and verifies it.
	ParseVerify(token string) (*Token, error)
}

// Token is a struct that holds the token and its claims.
type Token struct {
	jwt.RegisteredClaims
	Meta map[string]any `json:"meta,omitempty"`

	raw *jwt.Token
}

// Raw returns the raw token.
func (t *Token) Raw() *jwt.Token { return t.raw }

// Metadata returns the metadata of the token.
func (t *Token) Metadata() map[string]any { return t.Meta }

// NewTokenManager creates a new implementation of TokenManager based on JWT.
// It uses the given signer and verifier to sign and verify the token.
func NewTokenManager(signer jwt.Signer, verifier jwt.Verifier) *TokenManagerJWT {
	builder := jwt.NewBuilder(signer, jwt.WithContentType("jwt"))

	tm := TokenManagerJWT{
		builder:  builder,
		verifier: verifier,
	}

	return &tm
}

// TokenManagerJWT is an implementation of TokenManager based on JWT.
type TokenManagerJWT struct {
	builder  *jwt.Builder
	verifier jwt.Verifier
}

func (m *TokenManagerJWT) Sign(token *Token) (string, error) {
	t, err := m.builder.Build(token)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return t.String(), nil
}

func (m *TokenManagerJWT) Verify(token string) error {
	if _, err := m.ParseVerify(token); err != nil {
		return err
	}

	return nil
}

func (m *TokenManagerJWT) ParseVerify(token string) (*Token, error) {
	raw, err := jwt.Parse([]byte(token), m.verifier)
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token: %w", err))
	}

	t := Token{
		raw: raw,
	}

	if err := raw.DecodeClaims(&t); err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode claims: %w", err))
	}

	if !t.IsValidExpiresAt(time.Now()) {
		return nil, errors.Join(errkit.ErrTokenExpired, fmt.Errorf("token is expired"))
	}

	if !t.IsValidNotBefore(time.Now()) {
		return nil, errors.Join(errkit.ErrTokenNotBefore, fmt.Errorf("token is not valid yet"))
	}

	if !t.IsValidIssuedAt(time.Now()) {
		return nil, errors.Join(errkit.ErrTokenIssuedAt, fmt.Errorf("token is not valid at the current time"))
	}

	return &t, nil
}
