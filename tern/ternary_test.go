package tern

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestOP(t *testing.T) {
	type tcase[T any] struct {
		cond       bool
		t, f, want T
	}

	tests := map[string]tcase[string]{
		"true": {
			cond: true,
			t:    "true",
			f:    "false",
			want: "true",
		},

		"false": {
			cond: false,
			t:    "true",
			f:    "false",
			want: "false",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			td.Cmp(t, OP(tc.cond, tc.t, tc.f), tc.want)
		})
	}
}
