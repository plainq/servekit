package logkit

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	// ErrParseLevel indicates that string given to function ParseLevel can't be parsed to Level.
	ErrParseLevel Error = "string can't be parsed as Level, use: `error`, `warn`, `info`, `debug`"
)

// Error represents package level error related to logging work.
type Error string

func (e Error) Error() string { return string(e) }

// ParseLevel takes the string and tries to parse it to the Level.
func ParseLevel(lvl string) (slog.Level, error) {
	if lvl == "" {
		return slog.LevelInfo, ErrParseLevel
	}

	levels := map[string]slog.Level{
		strings.ToLower(slog.LevelWarn.String()):  slog.LevelWarn,
		strings.ToLower(slog.LevelError.String()): slog.LevelError,
		strings.ToLower(slog.LevelInfo.String()):  slog.LevelInfo,
		strings.ToLower(slog.LevelDebug.String()): slog.LevelDebug,
	}

	level, ok := levels[strings.ToLower(lvl)]
	if !ok {
		return slog.LevelInfo, fmt.Errorf("%s %w", lvl, ErrParseLevel)
	}

	return level, nil
}

// Options represents the configuration options for the logging library.
type Options struct {
	writer io.Writer
	level  *slog.LevelVar

	withSource bool
	withJSON   bool
}

// Option represents a function that modifies the configuration options for the logging library.
// The function takes a pointer to the Options struct and updates its fields accordingly.
// Example usage: WithWriter(w io.Writer) Option { return func(o *Options) { o.writer = w } }
type Option func(*Options)

// WithWriter changes the writer for each leveled loggers of StdLog to the given on.
func WithWriter(w io.Writer) Option { return func(o *Options) { o.writer = w } }

// WithLevel changes the underlying logging level of slog.Logger to the given on.
func WithLevel(level slog.Level) Option { return func(o *Options) { o.level.Set(level) } }

// WithJSON creates an Option that enables JSON formatting for log messages. When this Option is applied,
// log messages will be formatted as JSON objects.
// This Option modifies the 'withJSON' field of the Options struct.
// Example:
// options := &Options{}
// opt := WithJSON()
// opt(options)
// After applying the WithJSON Option, options.withJSON field will be set to true.
func WithJSON() Option { return func(o *Options) { o.withJSON = true } }

func New(options ...Option) *slog.Logger {
	o := Options{
		level:  &slog.LevelVar{},
		writer: os.Stderr,
	}

	for _, option := range options {
		option(&o)
	}

	var (
		handler        slog.Handler
		handlerOptions = slog.HandlerOptions{
			AddSource: o.withSource,
			Level:     o.level,
		}
	)

	switch o.withJSON {
	case true:
		handler = slog.NewJSONHandler(o.writer, &handlerOptions)

	default:
		handler = slog.NewTextHandler(o.writer, &handlerOptions)
	}

	return slog.New(handler)
}
