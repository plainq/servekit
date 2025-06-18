package jwtkit_test

import (
	"testing"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/maxatome/go-testdeep/td"
	"github.com/plainq/servekit/authkit/jwtkit"
	"github.com/plainq/servekit/errkit"
	"github.com/plainq/servekit/idkit"
)

func TestTokenManagerJWT(t *testing.T) {
	td.NewT(t)

	key := []byte("secret")
	signer, err := jwt.NewSignerHS(jwt.HS256, key)
	td.CmpNil(t, err)

	verifier, err := jwt.NewVerifierHS(jwt.HS256, key)
	td.CmpNil(t, err)

	manager := jwtkit.NewTokenManager(signer, verifier)

	t.Run("valid token", func(t *testing.T) {
		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ID:        idkit.XID(),
				Subject:   "test-subject",
				Audience:  []string{"test-audience"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
			},
			Meta: map[string]any{
				"foo": "bar",
			},
		}

		tokenStr, err := manager.Sign(token)
		td.CmpNil(t, err)

		parsedToken, err := manager.ParseVerify(tokenStr)
		td.CmpNil(t, err)

		td.Cmp(t, parsedToken.ID, token.ID)
		td.Cmp(t, parsedToken.Subject, token.Subject)
		td.Cmp(t, parsedToken.Audience, token.Audience)
		td.Cmp(t, parsedToken.Meta, td.JSON(`{"foo":"bar"}`))
		td.Cmp(t, parsedToken.ExpiresAt.Time.Unix(), token.ExpiresAt.Time.Unix())
		td.Cmp(t, parsedToken.NotBefore.Time.Unix(), token.NotBefore.Time.Unix())
		td.Cmp(t, parsedToken.IssuedAt.Time.Unix(), token.IssuedAt.Time.Unix())
	})

	t.Run("expired token", func(t *testing.T) {
		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			},
		}

		tokenStr, err := manager.Sign(token)
		td.CmpNil(t, err)

		_, err = manager.ParseVerify(tokenStr)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenExpired))
	})

	t.Run("token not before", func(t *testing.T) {
		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}

		tokenStr, err := manager.Sign(token)
		td.CmpNil(t, err)

		_, err = manager.ParseVerify(tokenStr)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenNotBefore))
	})

	t.Run("token issued at", func(t *testing.T) {
		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}

		tokenStr, err := manager.Sign(token)
		td.CmpNil(t, err)

		_, err = manager.ParseVerify(tokenStr)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenIssuedAt))
	})

	t.Run("verify method", func(t *testing.T) {
		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}

		tokenStr, err := manager.Sign(token)
		td.CmpNil(t, err)

		err = manager.Verify(tokenStr)
		td.CmpNil(t, err)
	})

	t.Run("invalid signature", func(t *testing.T) {
		key2 := []byte("wrong-secret")
		signer2, err := jwt.NewSignerHS(jwt.HS256, key2)
		td.CmpNil(t, err)
		manager2 := jwtkit.NewTokenManager(signer2, verifier)

		token := &jwtkit.Token{
			Claims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}

		tokenStr, err := manager2.Sign(token)
		td.CmpNil(t, err)

		_, err = manager.ParseVerify(tokenStr)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := manager.ParseVerify("a.b.c")
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})

	t.Run("custom claims", func(t *testing.T) {
		type CustomClaims struct {
			jwt.RegisteredClaims
			Foo string `json:"foo"`
		}

		claims := &CustomClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
			Foo: "bar",
		}

		builder := jwt.NewBuilder(signer)
		token, err := builder.Build(claims)
		td.CmpNil(t, err)

		parsedClaims := CustomClaims{}
		err = manager.ParseVerifyClaims(token.String(), &parsedClaims)
		td.CmpNil(t, err)
		td.Cmp(t, parsedClaims.Foo, "bar")
		td.Cmp(t, parsedClaims.ExpiresAt.Time.Unix(), claims.ExpiresAt.Time.Unix())
	})
}
