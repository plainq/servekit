// Package idkit provides the set of functions to generate
// different kind of identifiers.
package idkit

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/oklog/ulid/v2"
	"github.com/plainq/servekit/errkit"
	"github.com/rs/xid"
)

const (
	digiCodeMaxN = 9
	digiCodeLen  = 6
)

// NewULID returns ULID identifier as string.
func NewULID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to create ulid: %w", err)
	}

	return id.String(), nil
}

// ULID returns ULID identifier as string.
// More about ULID: https://github.com/ulid/spec
// Panics if it fails to generate an ID.
func ULID() string {
	id, err := NewULID()
	if err != nil {
		panic(fmt.Errorf("failed to generate ULID: %w", err))
	}
	return id
}

// ValidateULID validates string representation
// of ULID identifier.
func ValidateULID(id string) error {
	if _, err := ulid.Parse(id); err != nil {
		return errkit.ErrInvalidID
	}

	return nil
}

// XID returns short unique identifier as string.
func XID() string { return strings.ToUpper(xid.New().String()) }

// ValidateXID validates string representation of XID identifier.
func ValidateXID(id string) error {
	if _, err := xid.FromString(id); err != nil {
		return errkit.ErrInvalidID
	}

	return nil
}

// ParseXID returns XID identifier as object.
func ParseXID(id string) (xid.ID, error) {
	xID, err := xid.FromString(id)
	if err != nil {
		return xid.ID{}, errkit.ErrInvalidID
	}

	return xID, nil
}

// DigiCode returns 6-digit code as a string.
func DigiCode() string {
	var b strings.Builder
	for i := 0; i < digiCodeLen; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(digiCodeMaxN+1))
		if err != nil {
			panic(fmt.Errorf("failed to generate random digit: %w", err))
		}
		b.WriteString(strconv.FormatInt(n.Int64(), 10))
	}

	return b.String()
}

// ValidateDigiCode validates code from DigiCode.
func ValidateDigiCode(code string) error {
	if len(code) != digiCodeLen || utf8.RuneCountInString(code) != digiCodeLen {
		return errkit.ErrInvalidID
	}

	for _, r := range code {
		if !unicode.IsNumber(r) {
			return errkit.ErrInvalidID
		}
	}

	return nil
}
