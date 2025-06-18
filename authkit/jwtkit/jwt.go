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

	// ParseVerifyClaims takes a token string and parses and verifies it.
	// It decodes the claims into the provided claims struct.
	ParseVerifyClaims(token string, claims any) error
}

// Claims represents claims for JWT.
// See: https://tools.ietf.org/html/rfc7519#section-4.1
type Claims = jwt.RegisteredClaims

// Token represents claims for JWT with additional metadata.
type Token struct {
	Claims
	Meta map[string]any `json:"meta,omitempty"`

	raw *jwt.Token
}

// Raw returns the raw token.
func (t *Token) Raw() *jwt.Token { return t.raw }

// Metadata returns the metadata of the token.
func (t *Token) Metadata() map[string]any { return t.Meta }

// Validate validates the token claims.
func (t *Token) Validate(now time.Time) error {
	if !t.IsValidExpiresAt(now) {
		return errors.Join(errkit.ErrTokenExpired, errors.New("token is expired"))
	}

	if !t.IsValidNotBefore(now) {
		return errors.Join(errkit.ErrTokenNotBefore, errors.New("token is not valid yet"))
	}

	if !t.IsValidIssuedAt(now) {
		return errors.Join(errkit.ErrTokenIssuedAt, errors.New("token is not valid at the current time"))
	}

	return nil
}

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

	if err := t.Validate(time.Now()); err != nil {
		return nil, err
	}

	return &t, nil
}

func (m *TokenManagerJWT) ParseVerifyClaims(token string, claims any) error {
	raw, err := jwt.Parse([]byte(token), m.verifier)
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token: %w", err))
	}

	if err := raw.DecodeClaims(claims); err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode custom claims: %w", err))
	}

	t := Token{}
	if err := raw.DecodeClaims(&t); err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode standard claims: %w", err))
	}

	return t.Validate(time.Now())
}
