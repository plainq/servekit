package idkit

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestULID(t *testing.T) {
	t.Run("ULID", func(t *testing.T) {
		got := ULID()
		if len(got) != 26 {
			t.Errorf("the len of ULID() = %v, doesn't equal to 26 characters", got)
		}
	})
}

func TestValidateULID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		id := ULID()
		if err := ValidateULID(id); err != nil {
			t.Errorf("ULID should be valid but its not: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		id := "invalid id"
		if err := ValidateULID(id); err == nil || err != ErrInvalidID {
			t.Errorf("ULID should not be valid. Expected ErrInvalidID")
		}
	})
}

func TestDigiCode(t *testing.T) {
	t.Run("DigiCode", func(t *testing.T) {
		got := DigiCode()
		if len(got) != 6 || utf8.RuneCountInString(got) != 6 {
			t.Errorf("invalid digicode length: %d", len(got))
		}

		for _, r := range got {
			if !unicode.IsNumber(r) {
				t.Errorf("digicode contains char which is not a number: %s", string(r))
			}
		}
	})
}

func TestValidateDigiCode(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		code := DigiCode()
		if err := ValidateDigiCode(code); err != nil {
			t.Errorf("digicode should be valid but its not: %v", err)
		}
	})

	t.Run("Error len", func(t *testing.T) {
		id := "65789"
		if err := ValidateDigiCode(id); err == nil || err != ErrInvalidID {
			t.Errorf("digicode should not be valid. Expected ErrInvalidID")
		}
	})

	t.Run("Error invalid char", func(t *testing.T) {
		id := "65789C"
		if err := ValidateDigiCode(id); err == nil || err != ErrInvalidID {
			t.Errorf("digicode should not be valid. Expected ErrInvalidID")
		}
	})
}
