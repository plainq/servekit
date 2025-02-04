package hashkit

import (
	"reflect"
	"testing"
)

func TestNewBCryptHasher(t *testing.T) {
	type tcase struct {
		cost int
		want BCryptHasher
	}

	tests := map[string]tcase{
		"cost1":  {1, BCryptHasher{cost: 4}},
		"cost4":  {4, BCryptHasher{cost: 4}},
		"cost31": {31, BCryptHasher{cost: 31}},
		"cost32": {32, BCryptHasher{cost: 31}},
	}
	
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := NewBCryptHasher(WithCost(tc.cost)); !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("NewBCryptHasher() = %v, want %v", *got, tc.want)
			}
		})
	}
}