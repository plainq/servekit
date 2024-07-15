package respond

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/plainq/servekit/ctxkit"
	"github.com/plainq/servekit/errkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// errHTTPResponderInit is a guard to set the HTTPErrorResponder only once to avoid
	// accidentally reassigned errHTTPResponder which is used by default.
	errHTTPResponderInit sync.Once

	// errHTTPResponderInit is a guard to set the GRPCErrorResponder only once to avoid
	// accidentally reassigned errGRPCResponder which is used by default.
	errGRPCResponderInit sync.Once

	// htmlTemplaterInit is a guard to set the HTMLTemplateProvider only once to avoid
	// accidentally reassigned htmlTemplater which is used by default.
	htmlTemplaterInit sync.Once

	// htmlTemplater represents the default implementation of HTMLTemplateProvider.
	htmlTemplater HTMLTemplateProvider = &noopTemplater{}

	// errHTTPResponder represents the default implementation of HTTPErrorResponder func.
	errHTTPResponder HTTPErrorResponder = func(w http.ResponseWriter, err error) {
		if errors.Is(err, errkit.ErrAlreadyExists) {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		if errors.Is(err, errkit.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if errors.Is(err, errkit.ErrUnauthenticated) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		if errors.Is(err, errkit.ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, errkit.ErrInvalidArgument) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, errkit.ErrUnavailable) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	errGRPCResponder GRPCErrorResponder = func(err error) error {
		if errors.Is(err, errkit.ErrAlreadyExists) {
			return status.Error(codes.AlreadyExists, codes.AlreadyExists.String())
		}

		if errors.Is(err, errkit.ErrNotFound) {
			return status.Error(codes.NotFound, codes.NotFound.String())
		}

		if errors.Is(err, errkit.ErrUnauthenticated) {
			return status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		}

		if errors.Is(err, errkit.ErrUnauthorized) {
			return status.Error(codes.PermissionDenied, codes.PermissionDenied.String())
		}

		if errors.Is(err, errkit.ErrInvalidArgument) {
			return status.Error(codes.InvalidArgument, codes.InvalidArgument.String())
		}

		if errors.Is(err, errkit.ErrUnavailable) {
			return status.Error(codes.Unavailable, codes.Unavailable.String())
		}

		return status.Error(codes.Internal, codes.Internal.String())
	}
)

// HTTPErrorResponder represents a function which should be called to respond with an error on HTTP call.
type HTTPErrorResponder func(w http.ResponseWriter, err error)

// GRPCErrorResponder represents a function type that handles and responds to GRPC errors.
type GRPCErrorResponder func(err error) error

// HTMLTemplateProvider wraps a Template method to render requested HTML template.
type HTMLTemplateProvider interface {
	// Template renders the HTML templates by given name.
	Template(ctx context.Context, name string) (*template.Template, error)
}

// SetHTTPErrorResponder sets the given responder as errHTTPResponder.
func SetHTTPErrorResponder(responder HTTPErrorResponder) {
	errHTTPResponderInit.Do(func() { errHTTPResponder = responder })
}

// SetGRPCErrorResponder sets the given responder as errGRPCResponder.
func SetGRPCErrorResponder(responder GRPCErrorResponder) {
	errGRPCResponderInit.Do(func() { errGRPCResponder = responder })
}

// SetHTMLTemplateProvider sets the given htmlTemplater as htmlTemplater.
func SetHTMLTemplateProvider(templater HTMLTemplateProvider) {
	htmlTemplaterInit.Do(func() { htmlTemplater = templater })
}

// Status writes an HTTP status to the w http.ResponseWriter.
func Status(w http.ResponseWriter, _ *http.Request, statusCode int, options ...Option) {
	Options(w, options...)
	w.WriteHeader(statusCode)
}

// ErrorHTTP tries to map err to errkit.Error and based on result
// writes standard HTTP error with status statusCode to the response writer.
func ErrorHTTP(w http.ResponseWriter, r *http.Request, err error) {
	// Get log hook from the context to set an error which
	// will be logged along with access log line.
	if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
		hook(err)
	}

	// Call the default error responder.
	errHTTPResponder(w, err)
}

// ErrorGRPC tries to map err to errkit.Error and based on result
// writes standard gRPC error with status statusCode to the response writer.
func ErrorGRPC[T any](ctx context.Context, err error) (T, error) {
	// Get log hook from the context to set an error which
	// will be logged along with access log line.
	if hook := ctxkit.GetLogErrHook(ctx); hook != nil {
		hook(err)
	}

	// Call the default error responder.
	return zero[T](), errGRPCResponder(err)
}

// JSON tries to encode v into json representation and write it to response writer.
func JSON(w http.ResponseWriter, r *http.Request, v any, options ...Option) {
	o := Options(w, options...)

	coder := json.NewEncoder(w)
	coder.SetEscapeHTML(true)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(o.statusCode)

	if err := coder.Encode(v); err != nil {
		// Get log hook from the context to set an error which
		// will be logged along with access log line.
		if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
			hook(err)
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// HTML tries to encode v into json representation and write it to response writer.
func HTML(w http.ResponseWriter, r *http.Request, v []byte, options ...Option) {
	o := Options(w, options...)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(o.statusCode)

	// Return if v is empty.
	if len(v) == 0 {
		return
	}

	if _, err := w.Write(v); err != nil {
		// Get log hook from the context to set an error which
		// will be logged along with access log line.
		if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
			hook(err)
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func TemplateHTML(w http.ResponseWriter, r *http.Request, name string, v any, options ...Option) {
	o := Options(w, options...)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(o.statusCode)

	templ, err := htmlTemplater.Template(r.Context(), name)
	if err != nil {
		if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
			hook(err)
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := templ.Execute(w, v); err != nil {
		// Get log hook from the context to set an error which
		// will be logged along with access log line.
		if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
			hook(err)
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TEXT tries to write v to response writer.
func TEXT(w http.ResponseWriter, r *http.Request, v []byte, options ...Option) {
	o := Options(w, options...)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(o.statusCode)

	// Return if v is empty.
	if len(v) == 0 {
		return
	}

	if _, err := w.Write(v); err != nil {
		// Get log hook from the context to set an error which
		// will be logged along with access log line.
		if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
			hook(err)
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// ResponseOptions represents the options for an HTTP response.
type ResponseOptions struct {
	statusCode int
	headers    http.Header
}

// Options returns a pointer to a new ResponseOptions object with default values and applies the given options to it.
func Options(w http.ResponseWriter, options ...Option) *ResponseOptions {
	r := ResponseOptions{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
	}

	for _, option := range options {
		option(&r)
	}

	r.setHeadersToResponse(w)

	return &r
}

type Option func(o *ResponseOptions)

func WithStatus(status int) Option {
	return func(o *ResponseOptions) { o.statusCode = status }
}

func WithHeader(key, value string) Option {
	return func(o *ResponseOptions) { o.headers.Add(key, value) }
}

func (o *ResponseOptions) setHeadersToResponse(w http.ResponseWriter) {
	if len(o.headers) > 0 {
		for key, vals := range o.headers {
			for _, v := range vals {
				w.Header().Add(key, v)
			}
		}
	}
}

// zero returns default zeroed value for type T.
func zero[T any]() T {
	var v T
	return v
}

type noopTemplater struct{}

func (t *noopTemplater) Template(_ context.Context, _ string) (*template.Template, error) {
	return nil, fmt.Errorf("templater has not been initialized")
}
