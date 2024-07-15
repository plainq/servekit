package servekit

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestError_Error(t *testing.T) {
	f := func(got Error, want string) {
		t.Helper()

		td.Cmp(t, got.Error(), want)
	}

	f(ErrGracefullyShutdown, "shut down listener gracefully")
	f("test error", "test error")
}
