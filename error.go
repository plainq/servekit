package servekit

const (
	// ErrGracefullyShutdown represents an error message
	// indicating the failure to gracefully shut down a listener.
	ErrGracefullyShutdown Error = "shut down listener gracefully"
)

// Error represents a package level error. Implements builtin error interface.
type Error string

func (e Error) Error() string { return string(e) }
