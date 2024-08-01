package retry

import (
	"reflect"
	"testing"
	"time"
)

func TestNewConstantBackoff(t *testing.T) {
	want := &ConstantBackoff{
		minBackoffInterval: float64(50),
		maxBackoffInterval: float64(time.Second / time.Millisecond),
		maxJitterInterval:  float64(50),
	}

	if got := NewConstantBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond); !reflect.DeepEqual(got, want) {
		t.Errorf("NewConstantBackoff() = %v, want %v", got, want)
	}

	var _ Backoff = want
}

func Test_constantBackoff_Next(t *testing.T) {
	backoff := NewConstantBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)

	want := backoff.Next(1)

	if want < 50*time.Millisecond {
		t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
	}

	if want > time.Second+50*time.Millisecond {
		t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
	}
}

func TestNewLinearBackoff(t *testing.T) {
	want := &LinearBackoff{
		minBackoffInterval: float64(50),
		maxBackoffInterval: float64(time.Second / time.Millisecond),
		maxJitterInterval:  float64(50),
	}

	if got := NewLinearBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond); !reflect.DeepEqual(got, want) {
		t.Errorf("NewLinearBackoff() = %v, want %v", got, want)
	}

	var _ Backoff = want
}

func Test_linearBackoff_Next(t *testing.T) {
	backoff := NewLinearBackoff(50*time.Millisecond, time.Second, 50*time.Millisecond)

	t.Run("1 Retry", func(t *testing.T) {
		want := backoff.Next(1)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})

	t.Run("2 Retry", func(t *testing.T) {
		want := backoff.Next(2)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})

	t.Run("3 Retry", func(t *testing.T) {
		want := backoff.Next(3)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})
}

func TestNewExponentialBackoff(t *testing.T) {
	want := &ExponentialBackoff{
		exponentialFactor:  2,
		minBackoffInterval: float64(50),
		maxBackoffInterval: float64(time.Second / time.Millisecond),
		maxJitterInterval:  float64(50),
	}

	if got := NewExponentialBackoff(2, 50*time.Millisecond, time.Second, 50*time.Millisecond); !reflect.DeepEqual(got, want) {
		t.Errorf("NewExponentialBackoff() = %v, want %v", got, want)
	}

	var _ Backoff = want
}

func Test_exponentialBackoff_Next(t *testing.T) {
	backoff := NewExponentialBackoff(2, 50*time.Millisecond, time.Second, 50*time.Millisecond)

	t.Run("1 Retry", func(t *testing.T) {
		want := backoff.Next(1)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})

	t.Run("2 Retry", func(t *testing.T) {
		want := backoff.Next(2)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})

	t.Run("3 Retry", func(t *testing.T) {
		want := backoff.Next(3)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})

	t.Run("10 retry", func(t *testing.T) {
		want := backoff.Next(10)

		if want < 50*time.Millisecond {
			t.Errorf("backoff := %v snould not be less than %v ", want, 50*time.Millisecond)
		}

		if want > time.Second+50*time.Millisecond {
			t.Errorf("backoff := %v snould not be greater than %v", want, time.Second)
		}
	})
}
