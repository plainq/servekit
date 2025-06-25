package grpckit

import (
	"context"
	"errors"
	"sync"

	"github.com/plainq/servekit/ctxkit"
	"github.com/plainq/servekit/errkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// errGRPCResponderInit is a guard to set the GRPCErrorResponder only once to avoid
	// accidentally reassigned errGRPCResponder which is used by default.
	errGRPCResponderInit sync.Once

	errGRPCResponder GRPCErrorResponder = func(err error, options ...ResponseOption) error {
		o := NewResponseOptions(options...)

		if o.reportError {
			errkit.Report(err)
		}

		switch {
		case errors.Is(err, errkit.ErrAlreadyExists):
			return status.Error(codes.AlreadyExists, codes.AlreadyExists.String())

		case errors.Is(err, errkit.ErrNotFound):
			return status.Error(codes.NotFound, codes.NotFound.String())

		case errors.Is(err, errkit.ErrUnauthenticated):
			return status.Error(codes.Unauthenticated, codes.Unauthenticated.String())

		case errors.Is(err, errkit.ErrUnauthorized):
			return status.Error(codes.PermissionDenied, codes.PermissionDenied.String())

		case errors.Is(err, errkit.ErrInvalidArgument):
			return status.Error(codes.InvalidArgument, codes.InvalidArgument.String())

		case errors.Is(err, errkit.ErrUnavailable):
			return status.Error(codes.Unavailable, codes.Unavailable.String())

		default:
			return status.Error(codes.Internal, codes.Internal.String())
		}
	}
)

// GRPCErrorResponder represents a function type that handles errors from gRPC responses.
type GRPCErrorResponder func(err error, options ...ResponseOption) error

// SetGRPCErrorResponder sets the given responder as errGRPCResponder.
func SetGRPCErrorResponder(responder GRPCErrorResponder) {
	errGRPCResponderInit.Do(func() { errGRPCResponder = responder })
}

// ErrorGRPC tries to map err to errkit.Error and based on result
// writes standard gRPC error with status statusCode to the response writer.
func ErrorGRPC[T any](ctx context.Context, err error, options ...ResponseOption) (T, error) {
	// Get log hook from the context to set an error which
	// will be logged along with access log line.
	if hook := ctxkit.GetLogErrHook(ctx); hook != nil {
		hook(err)
	}

	return zero[T](), errGRPCResponder(err, options...)
}

// ResponseOptions represents the options for an gRPC response.
type ResponseOptions struct {
	statusCode  codes.Code
	reportError bool
}

// NewResponseOptions returns a pointer to a new ResponseOptions object with default values and applies the given options to it.
func NewResponseOptions(options ...ResponseOption) *ResponseOptions {
	r := ResponseOptions{
		statusCode:  codes.Unknown,
		reportError: false,
	}

	for _, option := range options {
		option(&r)
	}

	return &r
}

// ResponseOption represents a function type that modifies ResponseOptions for an gRPC response.
type ResponseOption func(o *ResponseOptions)

// WithStatus sets the given status code as the statusCode field of the ResponseOptions parameter.
func WithStatus(code codes.Code) ResponseOption {
	return func(o *ResponseOptions) {
		o.statusCode = code
	}
}

// zero returns default zeroed value for type T.
func zero[T any]() T {
	var v T

	return v
}
