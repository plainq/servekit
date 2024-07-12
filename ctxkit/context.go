package ctxkit

import (
	"context"
)

const (
	// LogErrHook represents a Key for context by which
	// the error log hook can be received from the context.
	logErrHook Key = "ctx.log-error-hook"

	// RequestID represents a Key for context by which
	// the request ID can be received from the context.
	requestID Key = "ctx.request-id"
)

// Key represents a context Key with custom type.
type Key string

// Set sets given value T to the context by given Key.
func Set[T any](ctx context.Context, key Key, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

// Get gets value of type T from context.
// If valued does not exist returns zeroed value for type T.
func Get[T any](ctx context.Context, key Key) T {
	value, ok := ctx.Value(key).(T)
	if !ok {
		return zero[T]()
	}

	return value
}

// SetLogErrHook sets the hook function to the context.
func SetLogErrHook(ctx context.Context, hook func(err error)) context.Context {
	return context.WithValue(ctx, logErrHook, hook)
}

// GetLogErrHook gets the hook function from the context which sets the error to server error log.
// If searched values is absent in context, then nil wil be returned.
func GetLogErrHook(ctx context.Context) func(error) {
	if hook, ok := ctx.Value(logErrHook).(func(error)); ok {
		return hook
	}

	return nil
}

// SetRequestID sets the request ID to the context.
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestID, id)
}

// GetRequestID gets the request ID from the context.
// If searched values is absent in context, then empty string wil be returned.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestID).(string); ok && id != "" {
		return id
	}

	return ""
}

// zero returns default zeroed value for type T.
func zero[T any]() (v T) { return v }
