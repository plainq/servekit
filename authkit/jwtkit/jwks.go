package jwtkit

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/plainq/servekit/errkit"
)

// Key represents a single key in a JWK set.
type Key struct {
	Use string `json:"use"`
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeyStore represents a set of keys from a JWKS endpoint.
type KeyStore struct {
	Keys []Key `json:"keys"`
}

// JWKSProvider is a token provider that uses a JWKS endpoint to verify tokens.
type JWKSProvider struct {
	mu         sync.RWMutex
	client     *http.Client
	jwksURL    string
	keyStore   *KeyStore
	keyCache   map[string]*rsa.PublicKey
	refreshInt time.Duration
	lastFetch  time.Time
}

// NewJWKSProvider creates a new JWKSProvider.
func NewJWKSProvider(jwksURL string, refreshInterval time.Duration) (*JWKSProvider, error) {
	p := &JWKSProvider{
		client:     &http.Client{Timeout: 10 * time.Second},
		jwksURL:    jwksURL,
		keyCache:   make(map[string]*rsa.PublicKey),
		refreshInt: refreshInterval,
	}

	if err := p.refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("initial jwks refresh: %w", err)
	}

	return p, nil
}

func (p *JWKSProvider) refresh(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastFetch) < p.refreshInt {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create jwks request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: unexpected status code: %d", resp.StatusCode)
	}

	var keyStore KeyStore
	if err := json.NewDecoder(resp.Body).Decode(&keyStore); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	p.keyStore = &keyStore
	p.keyCache = make(map[string]*rsa.PublicKey)

	for _, key := range keyStore.Keys {
		if key.Kty == "RSA" {
			pubKey, err := p.convertKey(key.E, key.N)
			if err != nil {
				continue
			}

			p.keyCache[key.Kid] = pubKey
		}
	}

	p.lastFetch = time.Now()

	return nil
}

func (*JWKSProvider) convertKey(e, n string) (*rsa.PublicKey, error) {
	decodedE, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	decodedN, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(decodedN),
		E: int(new(big.Int).SetBytes(decodedE).Int64()),
	}

	return pubKey, nil
}

func (p *JWKSProvider) getVerifier(kid string) (jwt.Verifier, error) {
	if err := p.refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("refresh jwks: %w", err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	pubKey, ok := p.keyCache[kid]
	if !ok {
		return nil, fmt.Errorf("kid '%s' not found in jwks", kid)
	}

	verifier, err := jwt.NewVerifierRS(jwt.RS256, pubKey)
	if err != nil {
		return nil, fmt.Errorf("create verifier: %w", err)
	}

	return verifier, nil
}

// ParseVerify parses and verifies a token using the key from the JWKS endpoint.
func (p *JWKSProvider) ParseVerify(token string) (*Token, error) {
	unverifiedToken, err := jwt.ParseNoVerify([]byte(token))
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token header: %w", err))
	}

	kid := unverifiedToken.Header().KeyID
	if kid == "" {
		return nil, errors.Join(errkit.ErrTokenInvalid, errors.New("missing kid in token header"))
	}

	verifier, err := p.getVerifier(kid)
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, err)
	}

	raw, err := jwt.Parse([]byte(token), verifier)
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

// ParseVerifyClaims parses and verifies a token using the key from the JWKS endpoint.
func (p *JWKSProvider) ParseVerifyClaims(token string, claims any) error {
	unverifiedToken, err := jwt.ParseNoVerify([]byte(token))
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token header: %w", err))
	}

	kid := unverifiedToken.Header().KeyID
	if kid == "" {
		return errors.Join(errkit.ErrTokenInvalid, errors.New("missing kid in token header"))
	}

	verifier, err := p.getVerifier(kid)
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, err)
	}

	raw, err := jwt.Parse([]byte(token), verifier)
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

// Verify verifies a token.
func (p *JWKSProvider) Verify(token string) error {
	if _, err := p.ParseVerify(token); err != nil {
		return err
	}

	return nil
}

// Sign is not supported for JWKSProvider.
func (*JWKSProvider) Sign(_ *Token) (string, error) {
	return "", errors.New("signing is not supported by JWKSProvider")
}
