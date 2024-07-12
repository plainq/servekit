package errkit

import (
	"errors"
	"sync"
)

var (
	reporter ErrorReporter
	register sync.Once
)

// ErrorReporter reports about errors.
type ErrorReporter interface {
	Report(err error)
}

// RegisterReporter registers given reporter as global reporter.
// Returns an error if the reporter has already been registered.
func RegisterReporter(r ErrorReporter) error {
	if reporter != nil {
		return errors.New("reporter already registered")
	}

	register.Do(func() { reporter = r })

	return nil
}

func Report(err error) {
	if reporter == nil {
		return
	}

	reporter.Report(err)
}
