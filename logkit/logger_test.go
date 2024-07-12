package logkit

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestWithWriter(t *testing.T) {
	f := func(input io.Writer) {
		t.Helper()

		options := Options{writer: nil}
		WithWriter(input)(&options)

		td.Cmp(t, options.writer, input)
	}

	f(&bytes.Buffer{})
	f(nil)
}

func TestParseLevel(t *testing.T) {
	f := func(input string, expLevel slog.Level, expErr error) {
		t.Helper()

		got, err := ParseLevel(input)
		td.Cmp(t, err, expErr)
		td.Cmp(t, got, expLevel)
	}

	f("", slog.LevelInfo, ErrParseLevel)
	f("Nonexistent level", slog.LevelInfo, fmt.Errorf("%s %w", "Nonexistent level", ErrParseLevel))
	f("warn", slog.LevelWarn, nil)
	f("error", slog.LevelError, nil)
	f("info", slog.LevelInfo, nil)
	f("debug", slog.LevelDebug, nil)
	f("wArn", slog.LevelWarn, nil)
}
