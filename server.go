package servekit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Listener is an interface that represents a listener which can serve requests.
// It requires the implementation of the Serve method that takes a context and returns an error.
type Listener interface {
	// Serve runs the listener, handling incoming requests. It takes a context as an argument
	// and returns an error if an error occurs while serving requests.
	Serve(ctx context.Context) error
}

// Server is a type that represents a server that holds a map of listeners.
type Server struct {
	logger *slog.Logger

	mu        sync.RWMutex
	listeners map[string]Listener
}

// NewServer creates a new Server instance with an empty listeners map
// and returns a pointer to the created Server.
func NewServer(logger *slog.Logger) *Server {
	s := Server{
		logger:    logger,
		listeners: make(map[string]Listener),
	}

	return &s
}

// RegisterListener adds a listener to the Server's listeners map.
func (s *Server) RegisterListener(name string, listener Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listeners[name] = listener

	s.logger.Info("Listener has been registered",
		slog.String("name", name),
	)
}

// Serve runs the server and serves requests from all listeners.
// It creates an error group and a listener context.
// It iterates through the listeners map and starts a goroutine for each listener.
// Each goroutine retries calling the listener's Serve method until it succeeds or the retry limit is reached.
// If the Serve method returns an error, it logs an error message and checks if the error is retryable.
// If the context is canceled, it returns the context error.
// If the retry limit is reached, it returns ErrRetryLimitReached.
// Finally, it waits for all goroutines to complete and returns any error encountered during serving.
func (s *Server) Serve(ctx context.Context) error {
	g, listenerCtx := errgroup.WithContext(ctx)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, listener := range s.listeners {
		g.Go(func() error {
			if err := listener.Serve(listenerCtx); err != nil {
				return fmt.Errorf("listener %s failed: %w", name, err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		s.logger.Error("Server failed",
			slog.String("error", err.Error()),
		)
	}

	return nil
}
