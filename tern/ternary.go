package tern

// OP represents generic ternary operator.
//
//nolint:revive // flag-parameter is ok here.
func OP[T any](cond bool, t, f T) T {
	if cond {
		return t
	}

	return f
}
