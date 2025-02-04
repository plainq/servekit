// Package idkit provides the set of functions to generate
// different kind of identifiers.
package idkit

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/oklog/ulid/v2"
	"github.com/plainq/servekit/errkit"
	"github.com/rs/xid"
	"github.com/valyala/fastrand"
)

const (
	digiCodeMaxN = 9
	digiCodeLen  = 6
)

// ULID returns ULID identifier as string.
// More about ULID: https://github.com/ulid/spec
func ULID() string {
	t := time.Now().UTC()
	e := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), e)

	return id.String()
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
	var (
		b   strings.Builder
		rng fastrand.RNG
	)

	rng.Seed(uint32(time.Now().UnixNano()))

	for i := 0; i < digiCodeLen; i++ {
		b.WriteString(strconv.Itoa(int(fastrand.Uint32n(digiCodeMaxN))))
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
