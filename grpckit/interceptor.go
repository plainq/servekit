package grpckit

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/plainq/servekit/ctxkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryInterceptor is an alias for grpc.UnaryServerInterceptor. It is used as a middleware
// to intercept unary RPC calls in a gRPC server.
type UnaryInterceptor = grpc.UnaryServerInterceptor

// StreamInterceptor is an alias for grpc.StreamServerInterceptor. It is used as a middleware
// to intercept streaming RPC calls in a gRPC server.
type StreamInterceptor = grpc.StreamServerInterceptor

// LoggingInterceptor is a gRPC unary server interceptor that logs method calls and their durations. It takes a logger
// instance as input and returns a UnaryServerInterceptor function.
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now().UTC()

		var reqErr error

		ctx = ctxkit.SetLogErrHook(ctx, func(err error) { reqErr = err })

		resp, err = handler(ctx, req)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				logger.Error("RPC",
					slog.String("code", s.Code().String()),
					slog.String("message", s.Message()),
					slog.String("method", info.FullMethod),
					slog.Duration("duration", time.Since(start)),
					slog.String("error", reqErr.Error()),
				)

				return resp, err
			}

			logger.Error("RPC",
				slog.String("method", info.FullMethod),
				slog.Duration("duration", time.Since(start)),
				slog.String("error", reqErr.Error()),
			)

			return resp, err
		}

		logger.Info("RPC",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", time.Since(start)),
		)

		return resp, err
	}
}

func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		code := 0

		resp, err = handler(ctx, req)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = int(s.Code())
			}
		}

		statusCode := strconv.Itoa(code)
		httpReqTotal := grpcReqTotalStr(info.FullMethod, statusCode)
		grpcReqDur := grpcReqDurationStr(info.FullMethod, statusCode)

		metrics.GetOrCreateCounter(httpReqTotal).
			Inc()

		metrics.GetOrCreateSummaryExt(grpcReqDur, 5*time.Minute, []float64{0.95, 0.99}).
			UpdateDuration(start)

		return resp, err
	}
}

func grpcReqDurationStr(route, code string) string {
	return `grpc_request_duration{route="` + route + `", code="` + code + `"}`
}

func grpcReqTotalStr(route, code string) string {
	return `grpc_requests_total{route="` + route + `", code="` + code + `"}`
}
