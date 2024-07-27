package httpkit

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/plainq/servekit/ctxkit"
)

// Middleware represents a function type that serves as a middleware in an HTTP server.
// It takes the next http.Handler as a parameter and returns an http.Handler.
// The middleware function is responsible for intercepting and processing HTTP requests and responses.
type Middleware = func(next http.Handler) http.Handler

func RecoveryMiddleware() Middleware           { return middleware.Recoverer }
func RedirectSlashesMiddleware() Middleware    { return middleware.RedirectSlashes }
func ProfilerMiddleware() http.Handler         { return middleware.Profiler() }
func CORSMiddleware(o cors.Options) Middleware { return cors.Handler(o) }

// LoggingMiddleware represents logging middleware.
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()

			var reqErr error

			ctx := ctxkit.SetLogErrHook(r.Context(), func(err error) { reqErr = err })

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()

			mwLogger := logger.With(
				slog.String("method", r.Method),
				slog.String("status", strconv.Itoa(status)),
				slog.String("route", r.RequestURI),
				slog.String("remote", r.RemoteAddr),
				slog.Duration("duration", time.Since(start)),
			)

			if status >= http.StatusInternalServerError {
				if reqErr != nil {
					mwLogger.Error(strconv.Itoa(status)+" "+http.StatusText(status),
						slog.String("error", reqErr.Error()),
					)

					return
				}

				mwLogger.Error(strconv.Itoa(status) + " " + http.StatusText(status))
			} else {
				mwLogger.Info(strconv.Itoa(status) + " " + http.StatusText(status))
			}
		}

		return http.HandlerFunc(fn)
	}
}

// MetricsMiddleware represents HTTP metrics collecting middlewares.
func MetricsMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := chi.RouteContext(r.Context())
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := strconv.Itoa(ww.Status())
			route := ctx.RoutePattern()

			httpReqDur := httpReqDurationStr(r.Method, route, status)
			httpReqTotal := httpReqTotalStr(r.Method, route, status)

			metrics.GetOrCreateSummaryExt(httpReqDur, 5*time.Minute, []float64{0.95, 0.99}).
				UpdateDuration(start)

			metrics.GetOrCreateCounter(httpReqTotal).
				Inc()
		}

		return http.HandlerFunc(fn)
	}
}

func httpReqDurationStr(method, route, status string) string {
	return `http_request_duration{method="` + method + `", route="` + route + `", code="` + status + `"}`
}

func httpReqTotalStr(method, route, status string) string {
	return `http_requests_total{method="` + method + `", route="` + route + `", code="` + status + `"}`
}
