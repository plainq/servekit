package hashkit

import (
	"errors"
	"fmt"

	"github.com/plainq/servekit/errkit"
	"golang.org/x/crypto/bcrypt"
)

// Hasher holds logic of hashing and checking the password.
type Hasher interface {
	// HashPassword takes password and return a hasher from it.
	HashPassword(pass string) (string, error)
	// CheckPassword takes password and perform comparison with
	// stored hashed password.
	CheckPassword(hash, pass string) error
}

// NewBCryptHasher returns a pointer to a new instance of BCryptHasher type.
func NewBCryptHasher(opts ...Option) *BCryptHasher {
	h := BCryptHasher{cost: bcrypt.DefaultCost}
	for _, opt := range opts {
		opt(&h)
	}
	return &h
}

type Option func(hasher *BCryptHasher)

// WithCost takes cost argument of type int and set the
// given value to 'BCryptHasher.cost' field.
// If provided cost exceed out of acceptable boundary
// then min or max cost wil be set.
func WithCost(cost int) Option {
	c := cost

	if c < bcrypt.MinCost {
		c = bcrypt.MinCost
	}

	if c > bcrypt.MaxCost {
		c = bcrypt.MaxCost
	}

	return func(h *BCryptHasher) {
		h.cost = c
	}
}

// BCryptHasher implements Hasher interface.
// Hashes passwords using bcrypt algorithm.
type BCryptHasher struct{ cost int }

func (*BCryptHasher) CheckPassword(hash, pass string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errkit.ErrPasswordIncorrect
		}

		return fmt.Errorf("password checking error: %w", err)
	}

	return nil
}

func (h *BCryptHasher) HashPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash: %w", err)
	}

	return string(hash), nil
}
