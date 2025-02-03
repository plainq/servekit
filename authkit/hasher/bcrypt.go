package hasher

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// NewBCryptHasher returns a pointer to a new instance
// of BCryptHasher type.
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
	if cost < bcrypt.MinCost {
		cost = bcrypt.MinCost
	}
	if cost > bcrypt.MaxCost {
		cost = bcrypt.MaxCost
	}
	return func(h *BCryptHasher) {
		h.cost = cost
	}
}

// BCryptHasher implements Hasher interface.
// Hashes passwords using bcrypt algorithm.
type BCryptHasher struct {
	cost int
}

func (h *BCryptHasher) CheckPassword(hash, pass string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return auth.ErrPasswordIncorrect
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
