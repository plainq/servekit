package sentrykit

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/plainq/servekit/errkit"
)

var (
	initClient sync.Once
	initErr    error
)

const (
	flushTimeout = 5 * time.Second
)

// Option configures Sentry.
type Option func(o *sentry.ClientOptions)

// WithEnv sets sentry.ClientOptions Environment field to track the name of the environment.
func WithEnv(env string) Option {
	return func(o *sentry.ClientOptions) { o.Environment = env }
}

func WithServerName(name string) Option {
	return func(o *sentry.ClientOptions) { o.ServerName = name }
}

func WithDist(dist string) Option {
	return func(o *sentry.ClientOptions) { o.Dist = dist }
}

func WithRelease(release string) Option {
	return func(o *sentry.ClientOptions) { o.Release = release }
}

func WithDebug(enabled bool) Option {
	return func(o *sentry.ClientOptions) { o.Debug = enabled }
}

func WithStacktrace(enabled bool) Option {
	return func(o *sentry.ClientOptions) { o.AttachStacktrace = enabled }
}

func WithSampleRate(rate float64) Option {
	return func(o *sentry.ClientOptions) { o.SampleRate = rate }
}

func WithTracing(enabled bool) Option {
	return func(o *sentry.ClientOptions) { o.EnableTracing = enabled }
}

func WithTracingSampleRate(rate float64) Option {
	return func(o *sentry.ClientOptions) { o.TracesSampleRate = rate }
}

func Init(dsn string, options ...Option) error {
	initClient.Do(func() {
		o := sentry.ClientOptions{
			Dsn:              dsn,
			Debug:            false,
			AttachStacktrace: false,
			EnableTracing:    false,
		}

		for _, option := range options {
			option(&o)
		}

		if err := sentry.Init(o); err != nil {
			initErr = fmt.Errorf("sentry: client initialization: %w", err)
		}

		if err := errkit.RegisterReporter(reporter{}); err != nil {
			initErr = fmt.Errorf("sentry: client registration: %w", err)
		}
	})

	return initErr
}

func Close() error {
	if !sentry.Flush(flushTimeout) {
		return errors.New("sentry: not flushed")
	}

	return nil
}

type reporter struct{}

func (reporter) Report(err error) { _ = sentry.CaptureMessage(err.Error()) }
