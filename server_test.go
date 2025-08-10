package servekit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

// mockListener implements the Listener interface for testing
type mockListener struct {
	serveFunc func(ctx context.Context) error
	name      string
}

func (m *mockListener) Serve(ctx context.Context) error {
	if m.serveFunc != nil {
		return m.serveFunc(ctx)
	}
	<-ctx.Done()
	return ErrGracefullyShutdown
}

func TestServer_GracefulShutdown(t *testing.T) {
	logger := slog.Default()
	server := NewServer(logger)

	// Add a mock listener that responds to context cancellation
	mockListener := &mockListener{
		name: "test-listener",
		serveFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return ErrGracefullyShutdown
		},
	}
	server.RegisterListener("test-listener", mockListener)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start serving in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()

	// Cancel after a short delay to trigger shutdown
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for server to shutdown
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected no error during graceful shutdown, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Server did not shutdown within expected time")
	}
}

func TestServer_ListenerError(t *testing.T) {
	logger := slog.Default()
	server := NewServer(logger)

	expectedError := errors.New("listener error")
	mockListener := &mockListener{
		serveFunc: func(ctx context.Context) error {
			return expectedError
		},
	}
	server.RegisterListener("failing-listener", mockListener)

	ctx := context.Background()
	err := server.Serve(ctx)

	if err == nil {
		t.Error("Expected error from failing listener, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected error to contain %v, got %v", expectedError, err)
	}
}

func TestServer_MultipleListeners(t *testing.T) {
	logger := slog.Default()
	server := NewServer(logger)

	// Add multiple listeners
	for i := 0; i < 3; i++ {
		listener := &mockListener{
			serveFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return ErrGracefullyShutdown
			},
		}
		server.RegisterListener(fmt.Sprintf("listener-%d", i), listener)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()

	// Cancel to trigger shutdown
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected no error during graceful shutdown with multiple listeners, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Server with multiple listeners did not shutdown within expected time")
	}
}

func TestServer_ShutdownMethod(t *testing.T) {
	logger := slog.Default()
	server := NewServer(logger)

	mockListener := &mockListener{
		serveFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return ErrGracefullyShutdown
		},
	}
	server.RegisterListener("test-listener", mockListener)

	// Test the dedicated Shutdown method
	err := server.Shutdown(1 * time.Second)
	if err != nil {
		t.Errorf("Expected no error from Shutdown method, got: %v", err)
	}
}
