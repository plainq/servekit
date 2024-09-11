package grpckit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/plainq/servekit"
	"github.com/plainq/servekit/logkit"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	// shutdownTimeout represents server default shutdown timeout.
	shutdownTimeout = 5 * time.Second
)

// Option implements functional options pattern for the ListenerGRPC type.
// Represents a function which receive a pointer to the generic struct that represents
// a part of ListenerGRPC configuration and changes it default values to the given ones.
//
// See the applyOptionsGRPC function to understand the configuration behaviour.
// Option functions should only be passed to ListenerGRPC constructor function NewListenerGRPC.
type Option[T ListenerConfig] func(o *T)

// WithLogger sets the server logger.
func WithLogger(logger *slog.Logger) Option[ListenerConfig] {
	return func(s *ListenerConfig) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithUnaryInterceptors is a function that takes a variable number of UnaryInterceptor functions
// and returns an Option[ListenerConfig]. This function is used to add UnaryInterceptors to the
// unaryInterceptors field of the ListenerConfig struct.
func WithUnaryInterceptors(interceptors ...UnaryInterceptor) Option[ListenerConfig] {
	return func(o *ListenerConfig) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors is a function that takes a variable number of StreamInterceptor functions
// and returns an Option[ListenerConfig]. This function is used to add StreamInterceptors to the
// streamInterceptors field of the ListenerConfig struct.
func WithStreamInterceptors(interceptors ...StreamInterceptor) Option[ListenerConfig] {
	return func(o *ListenerConfig) {
		o.streamInterceptors = append(o.streamInterceptors, interceptors...)
	}
}

// GRPCEndpointRegistrator abstracts a mechanics of registering
// the gRPC service in the gRPC server.
type GRPCEndpointRegistrator interface {
	Mount(server *grpc.Server)
}

// ListenerGRPC represents a struct that encapsulates a gRPC server listener.
type ListenerGRPC struct {
	logger   *slog.Logger
	listener net.Listener
	server   *grpc.Server
}

// NewListenerGRPC creates a new ListenerGRPC instance by creating a gRPC listener using a given address.
// It applies all the options to a default `applyOptionsGRPC` instance and sets the server options with
// the provided unary and stream interceptors. Finally, it returns the ListenerGRPC instance and potential error.
func NewListenerGRPC(addr string, options ...Option[ListenerConfig]) (*ListenerGRPC, error) {
	listener, grpcListenerErr := net.Listen("tcp", addr)
	if grpcListenerErr != nil {
		return nil, fmt.Errorf("create gRPC listener: %w", grpcListenerErr)
	}

	// Apply all option to the default applyOptionsHTTP.
	cfg := applyOptionsGRPC(options...)

	serverOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(cfg.unaryInterceptors...),
		grpc.ChainStreamInterceptor(cfg.streamInterceptors...),
	}

	l := ListenerGRPC{
		logger:   cfg.logger,
		listener: listener,
		server:   grpc.NewServer(serverOptions...),
	}

	return &l, nil
}

// Mount the given handlers to the listener gRPC server.
func (l *ListenerGRPC) Mount(handlers ...GRPCEndpointRegistrator) {
	for _, h := range handlers {
		h.Mount(l.server)
	}
}

func (l *ListenerGRPC) Serve(ctx context.Context) error {
	g, _ := errgroup.WithContext(ctx)

	// Handle graceful shutdown.
	g.Go(func() error { return l.handleShutdown(ctx) })

	g.Go(func() error {
		l.logger.Info("gRPC listener started to listen",
			slog.String("address", l.listener.Addr().String()),
		)

		if err := l.server.Serve(l.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("gRPC listener failed: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, servekit.ErrGracefullyShutdown) {
			panic(err)
		}

		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// handleShutdown blocks until select statement receives a signal from
// ctx.Done, after that new context.WithTimeout will be created and passed to
// http.Server Shutdown method.
//
// If Shutdown method returns non nil error, program will panic immediately.
func (l *ListenerGRPC) handleShutdown(ctx context.Context) error {
	<-ctx.Done()

	l.logger.Info("Shutting down the server!")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	g, _ := errgroup.WithContext(shutdownCtx)

	g.Go(func() error {
		done := make(chan struct{})

		go func() {
			l.server.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			return nil

		case <-shutdownCtx.Done():
			go l.server.Stop()
			return fmt.Errorf("shutdown gRPC listener: %w", shutdownCtx.Err())
		}
	})

	if err := g.Wait(); err != nil {
		l.logger.Error("Failed to shutdown the listener gracefully",
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("%w: %s", servekit.ErrGracefullyShutdown, err.Error())
	}

	return nil
}

func applyOptionsGRPC(options ...Option[ListenerConfig]) ListenerConfig {
	cfg := ListenerConfig{
		logger:             logkit.New(logkit.WithLevel(slog.LevelInfo)),
		unaryInterceptors:  make([]UnaryInterceptor, 0),
		streamInterceptors: make([]StreamInterceptor, 0),
	}

	for _, option := range options {
		option(&cfg)
	}

	return cfg
}

// ListenerConfig represents a struct that holds the configuration options for a gRPC server.
type ListenerConfig struct {
	logger             *slog.Logger
	unaryInterceptors  []UnaryInterceptor
	streamInterceptors []StreamInterceptor
}
