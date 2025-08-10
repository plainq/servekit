package servekit

const (
	// ErrGracefullyShutdown represents an error message
	// indicating the failure to gracefully shut down a listener.
	ErrGracefullyShutdown Error = "shut down listener gracefully"

	// ErrCertPathRequired represents an error message
	// indicating that a certificate file path is required.
	ErrCertPathRequired Error = "certificate file path is required"

	// ErrPrivateKeyPathRequired represents an error message
	// indicating that a private key file path is required.
	ErrPrivateKeyPathRequired Error = "private key file path is required"
)

// Error represents a package level error. Implements builtin error interface.
type Error string

// Error returns the error message as a string.
// Implements the error interface.
func (e Error) Error() string { return string(e) }
