package httpkit

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/plainq/servekit/retry"
)

const (
	// defaultNetDialTimeout represents default value
	// for net.Dialer Timeout field.
	defaultNetDialTimeout = 30 * time.Second

	// defaultKeepAliveTimeout represents default value
	// for net.Dialer KeepAlive field.
	defaultKeepAliveTimeout = 30 * time.Second

	// defaultTLSHandshakeTimeout represents default value
	// for http.Transport TLSHandshakeTimeout field.
	defaultTLSHandshakeTimeout = 5 * time.Second

	// defaultDisableKeepAlives represents default value
	// for http.Transport DisableKeepAlives field.
	defaultDisableKeepAlives = false

	// defaultMaxIdleConns represents default value
	// for http.Transport MaxIdleConns field.
	defaultMaxIdleConns = 100

	// defaultIdleConnTimeout represents default value
	// for http.Transport IdleConnTimeout field.
	defaultIdleConnTimeout = 90 * time.Second

	// defaultExpectContinueTimeout represents default value
	// for http.Transport ExpectContinueTimeout field.
	defaultExpectContinueTimeout = time.Second
)

var (
	// defaultMaxIdleConnsPerHost represents default value
	// for http.Transport MaxIdleConnsPerHost field.
	defaultMaxIdleConnsPerHost = runtime.GOMAXPROCS(0) + 1

	// redirectsErrRegExp represents regular expression which tries to match
	// text in error returned by net/http package when the configured number
	// of redirects is reached.
	redirectsErrRegExp = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// schemeErrRegExp represents regular expression which tries to match
	// text in error returned by net/http package when scheme specified in
	// the URL is invalid.
	schemeErrRegExp = regexp.MustCompile(`unsupported protocol scheme`)
)

// CustomDialer holds logic of establishing connection to remote network address.
type CustomDialer interface {
	// DialContext connects to the address on the named network using
	// the provided context.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Config holds configuration options which will be applied to http.Client.
type Config struct {
	// customDialer represents net.Dialer which is used to establish
	// connection with remote network address.
	// If customDialer is not nil it will be used instead of defaultDialer.
	customDialer CustomDialer

	// defaultDialer represents net.Dialer which is used to establish
	// connection with remote network address.
	defaultDialer *net.Dialer

	// retryConfig represents configuration for retry logic.
	retryBackoff     retry.Backoff
	retryMaxAttempts uint

	// disableKeepAlives if true, disables HTTP keep-alives and
	// will only use the connection to the server for a single
	// HTTP request. This is unrelated to the similarly named
	// TCP keep-alives.
	disableKeepAlives bool

	// tlsHandshakeTimeout controls the maximum amount of time to wait for a
	// TLS handshake to complete.
	tlsHandshakeTimeout time.Duration

	// maxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	maxIdleConns int

	// maxIdleConnsPerHost controls the maximum number of idle (keep-alive)
	// connections to keep per-host. Zero means no limit.
	maxIdleConnsPerHost int

	// idleConnTimeout controls the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	idleConnTimeout time.Duration

	// expectContinueTimeout controls the amount of time to wait for a server's
	// first response headers after fully writing the request headers if the
	// request has an Expect header.
	expectContinueTimeout time.Duration

	// responseHeaderTimeout controls the amount of time to wait for a server's
	// response headers after fully writing the request headers.
	responseHeaderTimeout time.Duration

	// writeBufferSize controls the size of the write buffer for the underlying
	// http.Transport.
	writeBufferSize int

	// readBufferSize controls the size of the read buffer for the underlying
	// http.Transport.
	readBufferSize int
}

func (c *Config) client() *http.Client {
	client := http.Client{
		Transport: c.transport(),
	}

	return &client
}

func (c *Config) transport() *http.Transport {
	transport := http.Transport{
		Proxy:                 c.proxy(),
		DialContext:           c.dialContext(),
		TLSHandshakeTimeout:   c.tlsHandshakeTimeout,
		DisableKeepAlives:     c.disableKeepAlives,
		MaxIdleConns:          c.maxIdleConns,
		MaxIdleConnsPerHost:   c.maxIdleConnsPerHost,
		IdleConnTimeout:       c.idleConnTimeout,
		ResponseHeaderTimeout: c.responseHeaderTimeout,
		ExpectContinueTimeout: c.expectContinueTimeout,
		WriteBufferSize:       c.writeBufferSize,
		ReadBufferSize:        c.readBufferSize,
	}

	return &transport
}

func (c *Config) dialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	if c.customDialer != nil {
		return c.customDialer.DialContext
	}

	return c.defaultDialer.DialContext
}

func (*Config) proxy() func(req *http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}

// ClientOption represents functional options pattern for Config type.
// ClientOption type represents a function which receive a pointer Config struct.
// ClientOption functions can only be passed to NewClient function.
// ClientOption function can change the default value of Config struct fields.
type ClientOption func(config *Config)

// WithTLSHandshakeTimeout sets timeout for TLS handshake to
// underlying http.Transport of http.Client, after which
// connection will be terminated.
func WithTLSHandshakeTimeout(timeout time.Duration) ClientOption {
	option := func(config *Config) {
		config.tlsHandshakeTimeout = timeout
	}

	return option
}

// WithCustomDialer sets given dialer as custom net.Dialer
// underlying http.Transport of http.Client.
func WithCustomDialer(dialer CustomDialer) ClientOption {
	option := func(config *Config) {
		config.customDialer = dialer
	}

	return option
}

// WithDialTimeout sets the Dial timeout to
// underlying http.Transport of http.Client.
func WithDialTimeout(timeout time.Duration) ClientOption {
	option := func(config *Config) {
		config.defaultDialer.Timeout = timeout
	}

	return option
}

// WithKeepAliveDisabled sets the DisableKeepAlives value to
// underlying http.Transport of http.Client.
func WithKeepAliveDisabled(disabled bool) ClientOption {
	option := func(config *Config) {
		config.disableKeepAlives = disabled
	}

	return option
}

// WithKeepAliveTimeout sets the KeepAlive timeout to
// underlying http.Transport of http.Client.
func WithKeepAliveTimeout(timeout time.Duration) ClientOption {
	option := func(config *Config) {
		config.defaultDialer.KeepAlive = timeout
	}

	return option
}

// WithMaxIdleConns sets the MaxIdleConns value to
// underlying http.Transport of http.Client.
func WithMaxIdleConns(maxn int) ClientOption {
	option := func(config *Config) {
		config.maxIdleConns = maxn
	}

	return option
}

// WithMaxIdleConnsPerHost sets the MaxIdleConnsPerHost value to
// underlying http.Transport of http.Client.
func WithMaxIdleConnsPerHost(maxn int) ClientOption {
	option := func(config *Config) {
		config.maxIdleConnsPerHost = maxn
	}

	return option
}

// WithResponseHeaderTimeout sets the ResponseHeaderTimeout value to
// underlying http.Transport of http.Client.
func WithResponseHeaderTimeout(timeout time.Duration) ClientOption {
	return func(config *Config) {
		config.responseHeaderTimeout = timeout
	}
}

// WithWriteBufferSize sets the WriteBufferSize value to
// underlying http.Transport of http.Client.
func WithWriteBufferSize(size int) ClientOption {
	return func(config *Config) {
		config.writeBufferSize = size
	}
}

// WithReadBufferSize sets the ReadBufferSize value to
// underlying http.Transport of http.Client.
func WithReadBufferSize(size int) ClientOption {
	option := func(config *Config) {
		config.readBufferSize = size
	}

	return option
}

// WithRetries configure http.Client to do retries
// when request failed and retry could be made.
func WithRetries(options ...retry.Option) ClientOption {
	retryCfg := retry.Options{}

	for _, option := range options {
		option(&retryCfg)
	}

	option := func(config *Config) {
		config.retryBackoff = retryCfg.Backoff()
		config.retryMaxAttempts = retryCfg.MaxRetries()
	}

	return option
}

// NewClient takes options to configure and return
// a pointer to a new instance of http.Client.
func NewClient(options ...ClientOption) *http.Client {
	var cfg = Config{
		// customDialer if not nil will be used instead of defaultDialer.
		customDialer: nil,

		// defaultDialer will be used if customDialer is nil.
		defaultDialer: &net.Dialer{
			Timeout:   defaultNetDialTimeout,
			KeepAlive: defaultKeepAliveTimeout,
		},

		disableKeepAlives:     defaultDisableKeepAlives,
		tlsHandshakeTimeout:   defaultTLSHandshakeTimeout,
		maxIdleConns:          defaultMaxIdleConns,
		maxIdleConnsPerHost:   defaultMaxIdleConnsPerHost,
		idleConnTimeout:       defaultIdleConnTimeout,
		expectContinueTimeout: defaultExpectContinueTimeout,
	}

	for _, option := range options {
		option(&cfg)
	}

	tripper := roundTripper{
		backoff: cfg.retryBackoff,
		client:  cfg.client(),
	}

	client := http.Client{
		Transport: &tripper,
	}

	return &client
}

// roundTripper implements http.RoundTripper interface using http.Client
// with tuned http.Transport and supports variety of resilience patterns
// like request hedging, circuit breaker, and retries with different
// backoff strategies.
type roundTripper struct {
	maxAttempts uint
	backoff     retry.Backoff
	client      *http.Client
}

//nolint:revive // cyclomatic is acceptable here.
func (t *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		attempts   = t.maxAttempts
		bodyReader io.ReadSeeker
	)

	if req.Body != nil {
		body, readBodyErr := io.ReadAll(req.Body)
		if readBodyErr != nil {
			return nil, readBodyErr
		}

		bodyReader = bytes.NewReader(body)

		// Here we set the io.NopCloser as request body
		// to prevent closing the body between retries.
		req.Body = io.NopCloser(bodyReader)
	}

	var res *http.Response

	for i := uint(0); i <= attempts; i++ {
		if res != nil {
			// In case of retry when the previous response is not nil we try to drain
			// the response body to utilize the HTTP connection.
			if err := res.Body.Close(); err != nil {
				return nil, fmt.Errorf("failed to close response body: %w", err)
			}
		}

		var doErr error
		res, doErr = t.client.Do(req)

		// If the bodyReader is not nil we try rewind the read position to the beginning
		// because it is already red at this point.
		if bodyReader != nil {
			if _, err := bodyReader.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("failed to rewind request body to the beggining: %w", err)
			}
		}

		// Here we check if received error represents url.Error which
		// in some cases can't be retried.
		if doErr != nil {
			var urlErr *url.Error
			if errors.As(doErr, &urlErr) {
				// If the error was occurred due to too many redirects
				// then we should not do the retry.
				if redirectsErrRegExp.MatchString(urlErr.Error()) {
					return nil, urlErr
				}

				// If the error was occurred due to an invalid protocol
				// scheme then we should not do the retry.
				if schemeErrRegExp.MatchString(urlErr.Error()) {
					return nil, urlErr
				}

				// if the error is related to TLS certificate
				// authority then we should not do the retry.
				var authorityErr x509.UnknownAuthorityError
				if errors.As(urlErr.Err, &authorityErr) {
					return nil, urlErr
				}
			}

			time.Sleep(t.backoff.Next(i))
			continue
		}

		// The '429 To Many Requests' and '503 Service Unavailable' are retryable status codes.
		// Here we check for 'Retry-After' response header that indicates when the target server
		// is ready to handle the client request.
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusServiceUnavailable {
			retryAfter := res.Header.Get("Retry-After")
			if retryAfter != "" {
				timeout, parseErr := strconv.ParseInt(retryAfter, 10, 64)
				// if the parseErr in not nil then we will just ignore the error
				// and will use default backoff.
				if parseErr == nil {
					time.Sleep(time.Second * time.Duration(timeout))
					continue
				}
			}

			time.Sleep(t.backoff.Next(i))
			continue
		}

		if res.StatusCode >= http.StatusInternalServerError &&
			res.StatusCode != http.StatusServiceUnavailable &&
			res.StatusCode != http.StatusNotImplemented {

			// Sleep before next retry.
			time.Sleep(t.backoff.Next(i))
			continue
		}
	}

	return res, nil
}
