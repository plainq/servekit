package logkit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
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

	timeFormat string
	withColor  bool
	withSource bool
	withJSON   bool
}

// Option represents a function that modifies the configuration options for the logging library.
type Option func(*Options)

// WithWriter changes the writer for each leveled loggers of StdLog to the given on.
func WithWriter(w io.Writer) Option { return func(o *Options) { o.writer = w } }

// WithLevel changes the underlying logging level of slog.Logger to the given on.
func WithLevel(level slog.Level) Option { return func(o *Options) { o.level.Set(level) } }

// WithJSON creates an Option that enables JSON formatting for log messages.
// When this Option is applied, log messages will be formatted as JSON objects.
func WithJSON() Option { return func(o *Options) { o.withJSON = true } }

// WithSource creates an Option that enables source code line number formatting for log messages.
// When this Option is applied, log messages will contain source code line number.
func WithSource() Option { return func(o *Options) { o.withSource = true } }

// WithColor creates an Option that enables color formatting for log messages.
// Will not take effect ff WithJSON is applied.
func WithColor() Option { return func(o *Options) { o.withColor = true } }

// WithTimeFormat creates an Option that change the time formatting for log messages.
func WithTimeFormat(format string) Option { return func(o *Options) { o.timeFormat = format } }

// New returns a pointer to a new instance of slog.Logger.
// Takes the variadic arguments of Option type to configure logger.
func New(options ...Option) *slog.Logger {
	o := Options{
		level:      &slog.LevelVar{},
		writer:     os.Stderr,
		timeFormat: time.DateTime,
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
		handler = tint.NewHandler(o.writer, &tint.Options{
			AddSource:  o.withSource,
			Level:      o.level,
			TimeFormat: o.timeFormat,
			NoColor:    o.withColor,
		})
	}

	return slog.New(handler)
}

// NewNop returns a new disabled logger that logs nothing.
func NewNop() *slog.Logger { return slog.New(&noopHandler{}) }

type noopHandler struct{}

func (noopHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (noopHandler) Handle(context.Context, slog.Record) error { return nil }
func (h noopHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h noopHandler) WithGroup(string) slog.Handler           { return h }
