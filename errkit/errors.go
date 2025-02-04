package errkit

// This is compiling time check for interface implementation.
var _ error = Error("") //nolint: errcheck // OK here.

const (
	// ErrNotFound indicates that requested entity can not be found.
	ErrNotFound Error = "not found"

	// ErrAlreadyExists indicates an attempt to create an entity
	// which is failed because such entity already exists.
	ErrAlreadyExists Error = "already exists"

	// ErrInvalidArgument indicates that client has specified an invalid argument.
	ErrInvalidArgument Error = "invalid argument"

	// ErrUnauthenticated indicates the request does not have valid
	// authentication credentials to perform the operation.
	ErrUnauthenticated Error = "authentication failed"

	// ErrUnauthorized indicates the caller does not have permission to
	// execute the specified operation. It must not be used if the caller
	// cannot be identified (use ErrUnauthenticated instead for those errors).
	ErrUnauthorized Error = "permission denied"

	// ErrUnavailable indicates that the service is currently unavailable.
	// This kind of error is retryable. Caller should retry with a backoff.
	ErrUnavailable Error = "temporarily unavailable"

	// ErrConnFailed shows that connection to a resource failed.
	ErrConnFailed Error = "connection failed"

	// ErrPasswordIncorrect indicates that the password is incorrect.
	ErrPasswordIncorrect Error = "password incorrect"

	// ErrTokenInvalid error means that given token is invalid or malformed.
	ErrTokenInvalid Error = "invalid token"

	// ErrTokenExpired error means that given token is already expired.
	ErrTokenExpired Error = "expired token"

	// ErrTokenNotBefore error means that given token is not valid yet.
	ErrTokenNotBefore Error = "token is not valid yet"

	// ErrTokenIssuedAt error means that given token is not valid at the current time.
	ErrTokenIssuedAt Error = "token is not valid at the current time"

	// ErrInvalidID represents an error which indicates that given TID is invalid.
	ErrInvalidID Error = "invalid identifier"

	// ErrValidation indicates that the data is not valid.
	ErrValidation Error = "validation failed"
)

// Error type represents package level errors.
type Error string

func (e Error) Error() string { return string(e) }
