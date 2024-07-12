package errkit

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestError_Error(t *testing.T) {
	f := func(name string, err error, want string) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			td.Cmp(t, err.Error(), want)
		})
	}

	f("ErrAlreadyExist", ErrAlreadyExists, "already exists")
	f("ErrInvalidArgument", ErrInvalidArgument, "invalid argument")
	f("ErrUnauthenticated", ErrUnauthenticated, "authentication failed")
	f("ErrUnauthorized", ErrUnauthorized, "permission denied")
	f("ErrUnavailable", ErrUnavailable, "temporarily unavailable")
	f("ErrConnFailed", ErrConnFailed, "connection failed")
	f("ErrNotFound", ErrNotFound, "not found")
	f("Custom", Error("test error"), "test error")
}
