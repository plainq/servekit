package jwtkit_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/maxatome/go-testdeep/td"
	"github.com/plainq/servekit/authkit/jwtkit"
	"github.com/plainq/servekit/idkit"
)

func TestJWKSProvider(t *testing.T) {
	td.NewT(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	td.CmpNil(t, err)

	kid := "test-kid"
	keyStore := jwtkit.KeyStore{
		Keys: []jwtkit.Key{
			{
				Use: "sig",
				Kty: "RSA",
				Kid: kid,
				Alg: "RS256",
				N:   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		td.CmpNil(t, json.NewEncoder(w).Encode(keyStore))
	}))
	t.Cleanup(server.Close)

	jwksProvider, err := jwtkit.NewJWKSProvider(server.URL, 1*time.Minute)
	td.CmpNil(t, err)

	signer, err := jwt.NewSignerRS(jwt.RS256, privateKey)
	td.CmpNil(t, err)

	builder := jwt.NewBuilder(signer, jwt.WithKeyID(kid))
	claims := &jwt.RegisteredClaims{
		ID:        idkit.XID(),
		Subject:   "test-subject",
		Audience:  []string{"test-audience"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token, err := builder.Build(claims)
	td.CmpNil(t, err)

	parsedToken, err := jwksProvider.ParseVerify(token.String())
	td.CmpNil(t, err)

	td.Cmp(t, parsedToken.Subject, "test-subject")
}

func ExampleNewJWKSProvider() {
	// This is a placeholder for a real JWKS endpoint
	// In a real application, you would use a URL like
	// "https://www.googleapis.com/oauth2/v3/certs"
	// or your own identity provider's JWKS endpoint.
	jwksURL := "http://127.0.0.1:8080/.well-known/jwks.json"

	// Create a new JWKSProvider with a 1-hour refresh interval.
	// The provider will fetch the keys from the URL upon creation
	// and then periodically refresh them.
	provider, err := jwtkit.NewJWKSProvider(jwksURL, 1*time.Hour)
	if err != nil {
		// In a real app, you would likely log this error and exit,
		// as the application cannot verify tokens without the keys.
		fmt.Printf("failed to create JWKS provider: %v", err)
		return
	}

	// The provider can now be used to verify tokens.
	// Typically, you would use this in a middleware to protect your routes.
	// For example:
	// token := "a.jwt.token"
	// parsedToken, err := provider.ParseVerify(token)
	_ = provider
}
