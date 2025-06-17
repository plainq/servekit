package retry

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/plainq/servekit/tern"
)

const (
	// ErrRetryLimitReached is an error indicating that the retry limit has been reached.
	ErrRetryLimitReached Error = "retry limit reached"
)

// Error represents package level errors.
type Error string

func (e Error) Error() string { return string(e) }

// RetryableError represents an error that can be retried. It encapsulates another error type
// and can be used to distinguish between errors that can be retried from those that cannot.
type RetryableError struct{ err error }

func (e RetryableError) Unwrap() error { return e.err }

func (e RetryableError) Error() string {
	if e.err == nil {
		return "retry: <nil>"
	}

	return "retry: " + e.err.Error()
}

// MarkRetryable marks error as retryable by wrapping it in RetryableError.
func MarkRetryable(err error) error {
	return tern.OP[error](err == nil, nil, &RetryableError{err: err})
}

// Backoff represents backoff logic.
type Backoff interface {
	// Next returns the timeout before next retry.
	Next(retry uint) time.Duration
}

// StaticBackoff represents a fixed duration as a backoff strategy.
type StaticBackoff time.Duration

func (b StaticBackoff) Next(uint) time.Duration { return time.Duration(b) }

// ExponentialBackoff implements Backoff interface where
// the backoff grows exponentially based on retry count.
type ExponentialBackoff struct {
	exponentialFactor  float64
	minBackoffInterval float64
	maxBackoffInterval float64
	maxJitterInterval  float64
}

// NewExponentialBackoff returns a pointer to a new instance of
// ExponentialBackoff struct, which implements Backoff interface.
func NewExponentialBackoff(f uint, minv, maxv, jitter time.Duration) *ExponentialBackoff {
	backoff := ExponentialBackoff{
		exponentialFactor:  float64(f),
		minBackoffInterval: float64(minv / time.Millisecond),
		maxBackoffInterval: float64(maxv / time.Millisecond),
		maxJitterInterval:  float64(jitter / time.Millisecond),
	}

	return &backoff
}

func (b *ExponentialBackoff) Next(retry uint) time.Duration {
	if retry <= 0 {
		return 0 * time.Millisecond
	}

	if b.minBackoffInterval >= b.maxBackoffInterval {
		return time.Duration(b.maxBackoffInterval) * time.Millisecond
	}

	mult := math.Pow(b.exponentialFactor, float64(retry))
	backoff := math.Min(b.minBackoffInterval*mult, b.maxBackoffInterval)
	jitter := float64(rand.Int64N(int64(b.maxJitterInterval))) //nolint:gosec

	return time.Duration(backoff+jitter) * time.Millisecond
}

// ConstantBackoff implements Backoff interface where the backoff is constant.
type ConstantBackoff struct {
	minBackoffInterval float64
	maxBackoffInterval float64
	maxJitterInterval  float64
}

// NewConstantBackoff returns a pointer to a new instance of
// ConstantBackoff struct, which implements Backoff interface.
func NewConstantBackoff(minv, maxv, jitter time.Duration) *ConstantBackoff {
	backoff := ConstantBackoff{
		minBackoffInterval: float64(minv / time.Millisecond),
		maxBackoffInterval: float64(maxv / time.Millisecond),
		maxJitterInterval:  float64(jitter / time.Millisecond),
	}

	return &backoff
}

func (b *ConstantBackoff) Next(retry uint) time.Duration {
	if retry <= 0 {
		return 0 * time.Millisecond
	}

	jitter := float64(rand.Int64N(int64(b.maxJitterInterval))) //nolint:gosec
	backoff := math.Min(b.minBackoffInterval, b.maxBackoffInterval)

	return time.Duration(backoff+jitter) * time.Millisecond
}

// LinearBackoff implements Backoff where the backoff grows linearly based on retry count.
type LinearBackoff struct {
	minBackoffInterval float64
	maxBackoffInterval float64
	maxJitterInterval  float64
}

// NewLinearBackoff returns a pointer to a new instance of
// ConstantBackoff struct, which implements Backoff interface.
func NewLinearBackoff(minv, maxv, jitter time.Duration) *LinearBackoff {
	backoff := LinearBackoff{
		minBackoffInterval: float64(minv / time.Millisecond),
		maxBackoffInterval: float64(maxv / time.Millisecond),
		maxJitterInterval:  float64(jitter / time.Millisecond),
	}

	return &backoff
}

func (b *LinearBackoff) Next(retry uint) time.Duration {
	if retry <= 0 {
		return 0 * time.Millisecond
	}

	jitter := float64(rand.Int64N(int64(b.maxJitterInterval))) //nolint:gosec
	backoff := math.Min(b.minBackoffInterval*float64(retry), b.maxBackoffInterval)

	return time.Duration(backoff+jitter) * time.Millisecond
}

// WithMaxAttempts is an Option function that sets the maximum number of attempts for a given operation.
// It takes a `maxAttempts` parameter of type `uint64` and updates the `maxRetries` field of the Options struct.
func WithMaxAttempts(maxAttempts uint) Option { return func(o *Options) { o.maxRetries = maxAttempts } }

// WithBackoff returns an Option function that sets the backoff strategy for retry logic.
// The Backoff implementation determines the timeout before the next retry based on the number of retries.
func WithBackoff(backoff Backoff) Option { return func(o *Options) { o.backoff = backoff } }

// WithLogger specifies the logger to be used for logging within the retry logic.
func WithLogger(logger *slog.Logger) Option { return func(o *Options) { o.logger = logger } }

// Options represents the configuration options for retry logic.
type Options struct {
	logger     *slog.Logger
	maxRetries uint
	backoff    Backoff
}

// MaxRetries returns number a max retry attempts.
func (o *Options) MaxRetries() uint { return o.maxRetries }

// Backoff returns the current implementation of Backoff interface.
func (o *Options) Backoff() Backoff { return o.backoff }

// Option is a function type that modifies the Options struct.
type Option func(*Options)

// Do retries the given function until it succeeds or the retry limit is reached.
// It takes a context and a function as parameters and optionally accepts additional options.
// The function is retried based on the specified options, which include the maximum number
// of retries and the backoff strategy.
// If the context is canceled, Do returns the context error.
// If the function returns an error that is not of type RetryableError, Do returns the error.
// If the retry limit is reached, Do returns ErrRetryLimitReached.
func Do(ctx context.Context, fn func(ctx context.Context) error, options ...Option) error {
	o := Options{
		maxRetries: 3,
		backoff:    StaticBackoff(0),
	}

	for _, option := range options {
		option(&o)
	}

	for i := uint(0); i < o.maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			if err := fn(ctx); err != nil {
				var rErr RetryableError

				if !errors.As(err, &rErr) {
					return err
				}

				backoff := o.backoff.Next(i)
				timer := time.NewTimer(backoff)

				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()

				case <-timer.C:
					timer.Stop()
					continue
				}
			}

			return nil
		}
	}

	return ErrRetryLimitReached
}
