package hashkit_test

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
	"github.com/plainq/servekit/authkit/hashkit"
	"github.com/plainq/servekit/errkit"
	"golang.org/x/crypto/bcrypt"
)

func TestNewBCryptHasher(t *testing.T) {
	td.NewT(t)

	t.Run("default cost", func(t *testing.T) {
		hasher := hashkit.NewBCryptHasher()
		td.Cmp(t, hasher, td.Struct(&hashkit.BCryptHasher{}, td.StructFields{
			"cost": bcrypt.DefaultCost,
		}))
	})

	t.Run("with cost", func(t *testing.T) {
		type tcase struct {
			cost int
			want int
		}

		tests := map[string]tcase{
			"cost below min": {bcrypt.MinCost - 1, bcrypt.MinCost},
			"cost at min":    {bcrypt.MinCost, bcrypt.MinCost},
			"cost above max": {bcrypt.MaxCost + 1, bcrypt.MaxCost},
			"cost at max":    {bcrypt.MaxCost, bcrypt.MaxCost},
			"cost in middle": {15, 15},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				hasher := hashkit.NewBCryptHasher(hashkit.WithCost(tc.cost))
				td.Cmp(t, hasher, td.Struct(&hashkit.BCryptHasher{}, td.StructFields{
					"cost": tc.want,
				}))
			})
		}
	})
}

func TestBCryptHasher_HashAndCheck(t *testing.T) {
	td.NewT(t)

	hasher := hashkit.NewBCryptHasher()
	password := "password123"

	hashedPassword, err := hasher.HashPassword(password)
	td.CmpNil(t, err)
	td.Cmp(t, hashedPassword, td.Not(""))

	t.Run("check correct password", func(t *testing.T) {
		err := hasher.CheckPassword(hashedPassword, password)
		td.CmpNil(t, err)
	})

	t.Run("check incorrect password", func(t *testing.T) {
		err := hasher.CheckPassword(hashedPassword, "wrongpassword")
		td.Cmp(t, err, errkit.ErrPasswordIncorrect)
	})

	t.Run("check invalid hash", func(t *testing.T) {
		err := hasher.CheckPassword("invalid-hash", password)
		td.Cmp(t, err, td.Not(errkit.ErrPasswordIncorrect))
		td.Cmp(t, err, td.NotNil())
	})
}
