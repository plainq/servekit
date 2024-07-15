package servekit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/heartwilltell/hc"
	"github.com/plainq/servekit/midkit"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// OptionGRPC implements functional options pattern for the ListenerGRPC type.
// Represents a function which receive a pointer to the generic struct that represents
// a part of ListenerGRPC configuration and changes it default values to the given ones.
//
// See the applyOptionsGRPC function to understand the configuration behaviour.
// OptionGRPC functions should only be passed to ListenerGRPC constructor function NewListenerGRPC.
type OptionGRPC[T grpcConfig] func(o *T)

// WithUnaryInterceptors is a function that takes a variable number of UnaryInterceptor functions
// and returns an OptionGRPC[grpcConfig]. This function is used to add UnaryInterceptors to the
// unaryInterceptors field of the grpcConfig struct.
func WithUnaryInterceptors(interceptors ...midkit.UnaryInterceptor) OptionGRPC[grpcConfig] {
	return func(o *grpcConfig) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors is a function that takes a variable number of StreamInterceptor functions
// and returns an OptionGRPC[grpcConfig]. This function is used to add StreamInterceptors to the
// streamInterceptors field of the grpcConfig struct.
func WithStreamInterceptors(interceptors ...midkit.StreamInterceptor) OptionGRPC[grpcConfig] {
	return func(o *grpcConfig) {
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
	health   hc.HealthChecker
	logger   *slog.Logger
	listener net.Listener
	server   *grpc.Server
}

// NewListenerGRPC creates a new ListenerGRPC instance by creating a gRPC listener using a given address.
// It applies all the options to a default `applyOptionsGRPC` instance and sets the server options with
// the provided unary and stream interceptors. Finally, it returns the ListenerGRPC instance and potential error.
func NewListenerGRPC(addr string, options ...OptionGRPC[grpcConfig]) (*ListenerGRPC, error) {
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
		if errors.Is(err, ErrGracefullyShutdown) {
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

		return fmt.Errorf("%w: %s", ErrGracefullyShutdown, err.Error())
	}

	return nil
}

func applyOptionsGRPC(options ...OptionGRPC[grpcConfig]) grpcConfig {
	cfg := grpcConfig{
		unaryInterceptors:  make([]midkit.UnaryInterceptor, 0),
		streamInterceptors: make([]midkit.StreamInterceptor, 0),
	}

	for _, option := range options {
		option(&cfg)
	}

	return cfg
}

// grpcConfig represents a struct that holds the configuration options for a gRPC server.
type grpcConfig struct {
	unaryInterceptors  []midkit.UnaryInterceptor
	streamInterceptors []midkit.StreamInterceptor
}
