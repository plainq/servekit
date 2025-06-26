package httpkit

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"sync"

	"github.com/plainq/servekit/ctxkit"
	"github.com/plainq/servekit/errkit"
)

var (
	// errHTTPResponderInit is a guard to set the HTTPErrorResponder only once to avoid
	// accidentally reassigned errHTTPResponder which is used by default.
	errHTTPResponderInit sync.Once

	// errHTTPResponder represents the default implementation of HTTPErrorResponder func.
	errHTTPResponder HTTPErrorResponder = func(w http.ResponseWriter, err error, options ...ResponseOption) {
		o := NewResponseOptions(w, options...)

		if o.reportError {
			errkit.Report(err)
		}

		var statusCode int

		switch {
		case errors.Is(err, errkit.ErrAlreadyExists):
			statusCode = http.StatusConflict

		case errors.Is(err, errkit.ErrNotFound):
			statusCode = http.StatusNotFound

		case errors.Is(err, errkit.ErrUnauthenticated):
			statusCode = http.StatusForbidden

		case errors.Is(err, errkit.ErrUnauthorized):
			statusCode = http.StatusUnauthorized

		case errors.Is(err, errkit.ErrInvalidArgument):
			statusCode = http.StatusBadRequest

		case errors.Is(err, errkit.ErrUnavailable):
			statusCode = http.StatusServiceUnavailable

		default:
			statusCode = http.StatusInternalServerError
		}

		if o.statusCode != 0 {
			statusCode = o.statusCode
		}

		http.Error(w, http.StatusText(statusCode), statusCode)
	}

	// htmlTemplaterInit is a guard to set the HTMLTemplateProvider only once to avoid
	// accidentally reassigned htmlTemplater which is used by default.
	htmlTemplaterInit sync.Once

	// htmlTemplater represents the default implementation of HTMLTemplateProvider.
	htmlTemplater HTMLTemplateProvider = &noopTemplater{}
)

// SetHTTPErrorResponder sets the given responder as errHTTPResponder.
func SetHTTPErrorResponder(responder HTTPErrorResponder) {
	errHTTPResponderInit.Do(func() { errHTTPResponder = responder })
}

// SetHTMLTemplater sets the given templater as htmlTemplater.
func SetHTMLTemplater(templater HTMLTemplateProvider) {
	htmlTemplaterInit.Do(func() { htmlTemplater = templater })
}

// HTMLTemplateProvider wraps a Template method to render requested HTML template.
type HTMLTemplateProvider interface {
	// Template renders the HTML templates by given name.
	Template(ctx context.Context, name string) (*template.Template, error)
}

// ResponseOption represents a function type that modifies ResponseOptions for an HTTP response.
type ResponseOption func(o *ResponseOptions)

// WithStatus sets the given status code as the statusCode field of the ResponseOptions parameter.
func WithStatus(code int) ResponseOption {
	return func(o *ResponseOptions) {
		o.statusCode = code
	}
}

// WithHeader is an Option function that adds the given key-value pair to the headers of the ResponseOptions.
// The headers are used to modify the headers of an HTTP response.
func WithHeader(key, value string) ResponseOption {
	return func(o *ResponseOptions) {
		o.headers.Add(key, value)
	}
}

// WithErrorReport is an Option function that will enable error reporting by the
// errkit.ErrorReporter. Used in the default implementation of HTTPErrorResponder.
func WithErrorReport() ResponseOption {
	return func(o *ResponseOptions) {
		o.reportError = true
	}
}

// ResponseOptions represents a set of options for an HTTP response.
type ResponseOptions struct {
	statusCode  int
	headers     http.Header
	reportError bool
}

// NewResponseOptions returns a pointer to a new ResponseOptions object with default values and applies the given options to it.
func NewResponseOptions(w http.ResponseWriter, options ...ResponseOption) *ResponseOptions {
	r := ResponseOptions{
		statusCode:  http.StatusOK,
		headers:     make(http.Header),
		reportError: false,
	}

	for _, option := range options {
		option(&r)
	}

	r.setHeadersToResponse(w)

	return &r
}

// setHeadersToResponse sets the headers of the ResponseOptions to the http.ResponseWriter.
func (o *ResponseOptions) setHeadersToResponse(w http.ResponseWriter) {
	if len(o.headers) > 0 {
		for key, vals := range o.headers {
			for _, v := range vals {
				w.Header().Add(key, v)
			}
		}
	}
}

// HTTPErrorResponder represents a function which should be called to respond with an error on HTTP call.
type HTTPErrorResponder func(w http.ResponseWriter, err error, options ...ResponseOption)

// Status writes an HTTP status to the w http.ResponseWriter.
func Status(w http.ResponseWriter, _ *http.Request, statusCode int, options ...ResponseOption) {
	o := NewResponseOptions(w, options...)
	o.statusCode = statusCode
	w.WriteHeader(o.statusCode)
}

// JSON tries to encode v into json representation and write it to response writer.
func JSON(w http.ResponseWriter, r *http.Request, v any, options ...ResponseOption) {
	o := NewResponseOptions(w, options...)

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
func HTML(w http.ResponseWriter, r *http.Request, v []byte, options ...ResponseOption) {
	o := NewResponseOptions(w, options...)

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

// TEXT tries to write v to response writer.
func TEXT(w http.ResponseWriter, r *http.Request, v []byte, options ...ResponseOption) {
	o := NewResponseOptions(w, options...)

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

// ErrorHTTP tries to map err to errkit.Error and based on result
// writes standard HTTP error with status statusCode to the response writer.
func ErrorHTTP(w http.ResponseWriter, r *http.Request, err error, options ...ResponseOption) {
	// Get log hook from the context to set an error which
	// will be logged along with access log line.
	if hook := ctxkit.GetLogErrHook(r.Context()); hook != nil {
		hook(err)
	}

	// Call the default error responder.
	errHTTPResponder(w, err, options...)
}

// TemplateHTML generates an HTML template response for the given name and data.
// It sets the Content-Type header to "text/html; charset=utf-8" and writes the
// response with the specified status code. If the template execution fails, an
// error will be logged and a 500 Internal Server Error response will be sent.
// Additional options can be passed to modify the response using the Option functions.
func TemplateHTML( //nolint:revive // argument-limit is acceptable here.
	w http.ResponseWriter, r *http.Request, name string, v any, options ...ResponseOption,
) {
	o := NewResponseOptions(w, options...)

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

type noopTemplater struct{}

func (*noopTemplater) Template(_ context.Context, _ string) (*template.Template, error) {
	return nil, errors.New("templater has not been initialized")
}
