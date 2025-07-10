package httpkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/heartwilltell/hc"
	"github.com/plainq/servekit"
	"github.com/plainq/servekit/ctxkit"
	"github.com/plainq/servekit/httpkit/statuspage"
	"github.com/plainq/servekit/logkit"
	"github.com/plainq/servekit/tern"
	"golang.org/x/sync/errgroup"
)

const (
	// readTimeout represents default read timeout for the http.Server.
	readTimeout = 2 * time.Second

	// readHeaderTimeout represents default read header timeout for the http.Server.
	readHeaderTimeout = 10 * time.Second

	// writeTimeout represents default write timeout for the http.Server.
	writeTimeout = 3 * time.Second

	// idleTimeout represents default idle timeout for the http.Server.
	idleTimeout = 10 * time.Second

	// shutdownTimeout represents server default shutdown timeout.
	shutdownTimeout = 5 * time.Second
)

// ListenerOptionConstraint represents a constraint for generic types
// that are related to ListenerOption.
type ListenerOptionConstraint interface {
	ListenerConfig | TimeoutsConfig | HealthConfig | MetricsConfig | PPROFConfig
}

// ListenerOption implements functional options pattern for the ListenerHTTP type.
// Represents a function which receive a pointer to the generic struct that represents
// a part of ListenerHTTP configuration and changes it default values to the given ones.
//
// See the applyOptionsHTTP function to understand the configuration behaviour.
// ListenerOption functions should only be passed to ListenerHTTP constructor function NewListenerHTTP.
type ListenerOption[T ListenerOptionConstraint] func(o *T)

// NewListenerOption returns a slice of ListenerOption[T].
func NewListenerOption[T ListenerOptionConstraint](options ...ListenerOption[T]) []ListenerOption[T] {
	o := make([]ListenerOption[T], 0, len(options))
	o = append(o, options...)
	return o
}

// WithTLS sets the TLS certificate and key to be used by the HTTP server.
// The certificate and key must be provided as strings containing the file paths.
// Note that this function is an ListenerOption for ListenerConfig and should be passed to the NewServer constructor.
func WithTLS(cert, key string) ListenerOption[ListenerConfig] {
	return func(c *ListenerConfig) {
		c.cert = cert
		c.key = key
	}
}

// WithGlobalMiddlewares sets given middlewares as router-wide middlewares.
// Means that they will be applied to each server endpoint.
func WithGlobalMiddlewares(middlewares ...Middleware) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		s.globalMiddlewares = append(s.globalMiddlewares, middlewares...)
	}
}

// WithHTTPServerTimeouts configures the HTTP listener TimeoutsConfig.
// Receives the following option to configure the endpoint:
// - HTTPServerReadHeaderTimeout - sets the http.Server ReadHeaderTimeout.
// - HTTPServerReadTimeout - sets the http.Server ReadTimeout.
// - HTTPServerWriteTimeout - sets the http.Server WriteTimeout.
// - HTTPServerIdleTimeout - sets the http.Server IdleTimeout.
func WithHTTPServerTimeouts(options ...ListenerOption[TimeoutsConfig]) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		for _, opt := range options {
			opt(&s.timeouts)
		}
	}
}

// HTTPServerReadHeaderTimeout sets the http.Server ReadHeaderTimeout.
func HTTPServerReadHeaderTimeout(t time.Duration) ListenerOption[TimeoutsConfig] {
	return func(c *TimeoutsConfig) { c.readHeaderTimeout = t }
}

// HTTPServerReadTimeout sets the http.Server ReadTimeout.
func HTTPServerReadTimeout(t time.Duration) ListenerOption[TimeoutsConfig] {
	return func(c *TimeoutsConfig) { c.readTimeout = t }
}

// HTTPServerWriteTimeout sets the http.Server WriteTimeout.
func HTTPServerWriteTimeout(t time.Duration) ListenerOption[TimeoutsConfig] {
	return func(c *TimeoutsConfig) { c.writeTimeout = t }
}

// HTTPServerIdleTimeout sets the http.Server IdleTimeout.
func HTTPServerIdleTimeout(t time.Duration) ListenerOption[TimeoutsConfig] {
	return func(c *TimeoutsConfig) { c.idleTimeout = t }
}

// WithLogger sets the server logger.
func WithLogger(logger *slog.Logger) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithHealthCheck turns on the health check endpoint.
// Receives the following option to configure the endpoint:
// - HealthChecker - to change the healthChecker implementation.
// - HealthCheckRoute - to set the endpoint route.
// - HealthCheckAccessLog - to enable access log for endpoint.
// - HealthCheckMetricsForEndpoint - to enable metrics collection for endpoint.
func WithHealthCheck(options ...ListenerOption[HealthConfig]) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		s.health.enable = true

		for _, opt := range options {
			opt(&s.health)
		}
	}
}

// HealthChecker represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.healthChecker.
func HealthChecker(checker hc.HealthChecker) ListenerOption[HealthConfig] {
	return func(c *HealthConfig) {
		// To not shoot in the leg. There are already a nop checker.
		if checker != nil {
			c.healthChecker = checker
		}
	}
}

// HealthCheckRoute represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.route.
func HealthCheckRoute(route string) ListenerOption[HealthConfig] {
	return func(c *HealthConfig) { c.route = route }
}

// HealthCheckAccessLog represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.accessLogsEnabled to true.
func HealthCheckAccessLog(enable bool) ListenerOption[HealthConfig] {
	return func(c *HealthConfig) { c.accessLogsEnabled = enable }
}

// HealthCheckMetricsForEndpoint represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.metricsForEndpointEnabled to true.
func HealthCheckMetricsForEndpoint(enable bool) ListenerOption[HealthConfig] {
	return func(c *HealthConfig) { c.metricsForEndpointEnabled = enable }
}

// HealthCheckReportJSON represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.healthReport to healthReportJSON.
func HealthCheckReportJSON() ListenerOption[HealthConfig] {
	return func(c *HealthConfig) { c.healthReport = healthReportJSON }
}

// HealthCheckReportHTML represents an optional function for WithHealthCheck function.
// If passed to the WithHealthCheck, will set the ServerSettings.health.healthReport to healthReportHTML.
func HealthCheckReportHTML() ListenerOption[HealthConfig] {
	return func(c *HealthConfig) { c.healthReport = healthReportHTML }
}

// WithMetrics turns on the metrics endpoint.
// Receives the following option to configure the endpoint:
// - MetricsRoute - to set the endpoint route.
// - MetricsAccessLog - to enable access log for endpoint.
// - MetricsMetricsForEndpoint - to enable metrics collection for endpoint.
func WithMetrics(options ...ListenerOption[MetricsConfig]) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		s.metrics.enable = true

		for _, opt := range options {
			opt(&s.metrics)
		}
	}
}

// MetricsRoute represents an optional function for WithMetrics function.
// If passed to the WithMetrics, will set the ServerSettings.health.route.
func MetricsRoute(route string) ListenerOption[MetricsConfig] {
	return func(c *MetricsConfig) { c.route = route }
}

// MetricsAccessLog represents an optional function for WithMetrics function.
// If passed to the WithMetrics, will set the ServerSettings.health.accessLogsEnabled to true.
func MetricsAccessLog(enable bool) ListenerOption[MetricsConfig] {
	return func(c *MetricsConfig) { c.accessLogsEnabled = enable }
}

// MetricsMetricsForEndpoint represents an optional function for WithMetrics function.
// If passed to the WithMetrics, will set the ServerSettings.health.metricsForEndpointEnabled to true.
func MetricsMetricsForEndpoint(enable bool) ListenerOption[MetricsConfig] {
	return func(c *MetricsConfig) { c.metricsForEndpointEnabled = enable }
}

// WithProfiler turns on the profiler endpoint.
func WithProfiler(cfg PPROFConfig) ListenerOption[ListenerConfig] {
	return func(s *ListenerConfig) {
		s.profiler.enable = true
		s.profiler.accessLogsEnabled = cfg.accessLogsEnabled

		if cfg.route != "" {
			s.profiler.route = cfg.route
		}
	}
}

type ListenerHTTP struct {
	enableTLS bool
	cert, key string

	health hc.HealthChecker
	logger *slog.Logger

	router chi.Router
	server *http.Server
}

// NewListenerHTTP creates a new ListenerHTTP with the specified address and options.
// The options parameter is a variadic argument that accepts functions of type ListenerOption.
// The ListenerHTTP instance is returned, which can be used to mount routes and start serving requests.
func NewListenerHTTP(addr string, options ...ListenerOption[ListenerConfig]) (*ListenerHTTP, error) {
	router := chi.NewRouter()

	l := ListenerHTTP{
		router: router,
		server: &http.Server{ //nolint: gosec // OK here. Timeouts will be set later.
			Addr:    addr,
			Handler: router,
		},
	}

	// Apply all option to the default applyOptionsHTTP.
	cfg := applyOptionsHTTP(options...)

	// Set listener logger.
	l.logger = cfg.logger

	if l.enableTLS {
		if err := l.configureTLS(cfg); err != nil {
			return nil, fmt.Errorf("configure TLS: %w", err)
		}
	}

	// Use global middlewares.
	l.router.Use(cfg.globalMiddlewares...)

	if err := l.configureHealth(cfg); err != nil {
		return nil, fmt.Errorf("configure health: %w", err)
	}

	if err := l.configureMetrics(cfg); err != nil {
		return nil, fmt.Errorf("configure metrics: %w", err)
	}

	if err := l.configureProfiler(cfg); err != nil {
		return nil, fmt.Errorf("configure profiler: %w", err)
	}

	return &l, nil
}

func (l *ListenerHTTP) MountGroup(route string, fn func(r chi.Router)) {
	l.router.Route(route, fn)
}

func (l *ListenerHTTP) Mount(route string, handler http.Handler, middlewares ...Middleware) {
	l.router.Route(route, func(r chi.Router) {
		r.Use(middlewares...)
		r.Mount("/", handler)
	})
}

func (l *ListenerHTTP) Serve(ctx context.Context) error {
	if l.server.Addr == "" {
		return fmt.Errorf("invalid listener address: %s", l.server.Addr)
	}

	g, serveCtx := errgroup.WithContext(ctx)

	// Handle shutdown signal in the background.
	g.Go(func() error { return l.handleShutdown(serveCtx) })

	g.Go(func() error {
		protocol := tern.OP(l.enableTLS, "HTTPS", "HTTP")

		l.logger.Info(protocol+" listener started to listen",
			slog.String("address", l.server.Addr),
		)

		if err := l.serveFunc(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listener failed: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, servekit.ErrGracefullyShutdown) {
			l.logger.Error("Failed to shutdown the listener gracefully",
				slog.String("address", l.server.Addr),
				slog.String("error", err.Error()),
			)
		} else {
			l.logger.Error("Listener failed to serve",
				slog.String("address", l.server.Addr),
				slog.String("error", err.Error()),
			)
		}

		return err
	}

	return nil
}

func (l *ListenerHTTP) serveFunc() error {
	switch {
	case l.enableTLS:
		return l.server.ListenAndServeTLS(l.cert, l.key)

	default:
		return l.server.ListenAndServe()
	}
}

func (l *ListenerHTTP) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if l.health == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := l.health.Health(r.Context()); err != nil {
		ctxkit.GetLogErrHook(r.Context())(err)

		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (l *ListenerHTTP) healthCheckHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if err := l.health.Health(r.Context()); err != nil {
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"status":  "503 Service Unavailable",
			"message": "Service is temporarily unavailable. Please try again later.",
		}); encodeErr != nil {
			ctxkit.GetLogErrHook(r.Context())(errors.Join(err, encodeErr))

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		ctxkit.GetLogErrHook(r.Context())(err)

		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "200 OK",
		"message": "Service is healthy",
	}); err != nil {
		ctxkit.GetLogErrHook(r.Context())(err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (l *ListenerHTTP) healthCheckHandlerHTML(w http.ResponseWriter, r *http.Request) {
	var (
		healthErr = l.health.Health(r.Context())
		report    = hc.NewServiceReport()
	)

	if shc, ok := l.health.(*hc.MultiServiceChecker); ok {
		report = shc.Report()
	}

	var buf bytes.Buffer

	if err := statuspage.RenderStatus(&buf, report); err != nil {
		ctxkit.GetLogErrHook(r.Context())(errors.Join(healthErr, fmt.Errorf("render status page: %w", err)))

		l.logger.Error("Failed to render status page",
			slog.String("error", err.Error()),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, &buf); err != nil {
		ctxkit.GetLogErrHook(r.Context())(errors.Join(
			healthErr,
			fmt.Errorf("write status page buffer to response writer: %w", err),
		))

		l.logger.Error("Failed to write status page buffer to response writer",
			slog.String("error", err.Error()),
		)

		return
	}
}

func (*ListenerHTTP) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	metrics.WritePrometheus(w, true)
}

func (l *ListenerHTTP) handleShutdown(ctx context.Context) error {
	<-ctx.Done()

	l.logger.Info("Shutting down the listener",
		slog.String("address", l.server.Addr),
	)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := l.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("%w: %v", servekit.ErrGracefullyShutdown, err)
	}

	return nil
}

// ListenerConfig holds ListenerHTTP configuration.
type ListenerConfig struct {
	cert, key string

	// logger represents a logger for HTTP server.
	logger *slog.Logger

	// timeouts holds an HTTP server timeouts configuration.
	timeouts TimeoutsConfig

	// globalMiddlewares holds a set of router-wide HTTP middlewares,
	// which are applied to each endpoint.
	globalMiddlewares []Middleware

	// health holds configuration of health endpoint.
	health HealthConfig

	// metrics holds configuration for metrics endpoint.
	metrics MetricsConfig

	// profiler holds configuration fot profiler endpoint.
	profiler PPROFConfig
}

func applyOptionsHTTP(options ...ListenerOption[ListenerConfig]) ListenerConfig {
	cfg := ListenerConfig{
		logger: logkit.New(logkit.WithLevel(slog.LevelInfo)),

		timeouts: TimeoutsConfig{
			readHeaderTimeout: readHeaderTimeout,
			readTimeout:       readTimeout,
			writeTimeout:      writeTimeout,
			idleTimeout:       idleTimeout,
		},

		globalMiddlewares: []Middleware{},

		health: HealthConfig{
			healthChecker:             hc.NewNopChecker(),
			enable:                    false,
			accessLogsEnabled:         false,
			metricsForEndpointEnabled: false,
			route:                     "/health",
		},

		metrics: MetricsConfig{
			enable:                    false,
			accessLogsEnabled:         false,
			metricsForEndpointEnabled: false,
			route:                     "/metrics",
		},

		profiler: PPROFConfig{
			enable:            false,
			accessLogsEnabled: false,
			route:             "/debug",
		},
	}

	for _, option := range options {
		option(&cfg)
	}

	return cfg
}

func (l *ListenerHTTP) configureTLS(cfg ListenerConfig) error {
	if cfg.cert == "" {
		return servekit.ErrCertPathRequired
	}

	if cfg.key == "" {
		return servekit.ErrPrivateKeyPathRequired
	}

	l.enableTLS = true
	l.cert = cfg.cert
	l.key = cfg.key

	return nil
}

func (l *ListenerHTTP) configureHealth(cfg ListenerConfig) error {
	if cfg.health.enable {
		if cfg.health.healthChecker != nil {
			l.health = cfg.health.healthChecker
		}

		if cfg.health.route == "" {
			return errors.New("empty health route")
		}

		if !strings.HasPrefix(cfg.health.route, "/") {
			return fmt.Errorf(
				"invalid health route: %q (route should start with '/' slash)",
				cfg.health.route,
			)
		}

		l.router.Route(cfg.health.route, func(health chi.Router) {
			if cfg.health.accessLogsEnabled {
				health.Use(LoggingMiddleware(l.logger))
			}

			if cfg.health.metricsForEndpointEnabled {
				health.Use(MetricsMiddleware())
			}

			switch cfg.health.healthReport {
			case healthReportJSON:
				health.Get("/", l.healthCheckHandlerJSON)
				health.Head("/", l.healthCheckHandler)

			case healthReportHTML:
				health.Get("/", l.healthCheckHandlerHTML)
				health.Head("/", l.healthCheckHandler)

			default:
				health.Get("/", l.healthCheckHandler)
				health.Head("/", l.healthCheckHandler)
			}
		})
	}

	return nil
}

func (l *ListenerHTTP) configureMetrics(cfg ListenerConfig) error {
	if cfg.metrics.enable {
		if cfg.metrics.route == "" {
			return errors.New("empty metrics route")
		}

		if !strings.HasPrefix(cfg.metrics.route, "/") {
			return fmt.Errorf("invalid metrics route: %q (route should start with '/' slash)",
				cfg.metrics.route,
			)
		}

		l.router.Route(cfg.metrics.route, func(metrics chi.Router) {
			if cfg.metrics.accessLogsEnabled {
				metrics.Use(LoggingMiddleware(l.logger))
			}

			if cfg.metrics.metricsForEndpointEnabled {
				metrics.Use(MetricsMiddleware())
			}

			metrics.Get("/", l.metricsHandler)
		})
	}

	return nil
}

func (l *ListenerHTTP) configureProfiler(cfg ListenerConfig) error {
	if cfg.profiler.enable {
		if cfg.profiler.route == "" {
			return errors.New("empty profiler route")
		}

		if !strings.HasPrefix(cfg.profiler.route, "/") {
			return fmt.Errorf(
				"invalid profiler route: %q (route should start with '/' slash)",
				cfg.profiler.route,
			)
		}

		l.router.Route(cfg.profiler.route, func(profiler chi.Router) {
			if cfg.profiler.accessLogsEnabled {
				profiler.Use(LoggingMiddleware(l.logger))
			}

			profiler.Mount("/", middleware.Profiler())
		})
	}

	return nil
}

// TimeoutsConfig holds an HTTP server TimeoutsConfig configuration.
type TimeoutsConfig struct {
	// readTimeout represents the http.Server ReadTimeout.
	readTimeout time.Duration

	// readHeaderTimeout represents the http.Server ReadHeaderTimeout.
	readHeaderTimeout time.Duration

	// writeTimeout represents the http.Server WriteTimeout.
	writeTimeout time.Duration

	// idleTimeout represents the http.Server IdleTimeout.
	idleTimeout time.Duration
}

// HealthConfig represents configuration for builtin health check route.
type HealthConfig struct {
	enable                    bool
	accessLogsEnabled         bool
	metricsForEndpointEnabled bool
	route                     string
	healthChecker             hc.HealthChecker
	healthReport              healthReport
}

// healthReport represents a type for health report format.
type healthReport int8

// healthReport constants.
const (
	healthReportNone healthReport = iota
	healthReportJSON
	healthReportHTML
)

// MetricsConfig represents configuration for builtin metrics route.
type MetricsConfig struct {
	enable                    bool
	accessLogsEnabled         bool
	metricsForEndpointEnabled bool
	route                     string
}

// PPROFConfig represents configuration for builtin profiler route.
type PPROFConfig struct {
	enable            bool
	accessLogsEnabled bool
	route             string
}
