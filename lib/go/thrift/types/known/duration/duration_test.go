package duration

import (
	"math"
	"reflect"
	"testing"
	"time"
)

const (
	minGoSeconds = math.MinInt64 / int64(1e9)
	maxGoSeconds = math.MaxInt64 / int64(1e9)
	absSeconds   = 315576000000 // 10000yr * 365.25day/yr * 24hr/day * 60min/hr * 60sec/min
)

func TestDuration(t *testing.T) {
	for _, tt := range []struct {
		in   time.Duration
		want *Duration
	}{
		{in: time.Duration(0), want: &Duration{}},
		{in: -time.Second, want: &Duration{Seconds: -1}},
		{in: time.Second, want: &Duration{Seconds: 1}},
		{
			in:   -time.Second - time.Millisecond,
			want: &Duration{Seconds: -1, Nanos: -1e6},
		},
		{
			in:   +time.Second + time.Millisecond,
			want: &Duration{Seconds: 1, Nanos: 1e6},
		},
		{
			in:   time.Duration(math.MinInt64),
			want: &Duration{Seconds: minGoSeconds, Nanos: int32(math.MinInt64 - 1e9*minGoSeconds)},
		},
		{
			in:   time.Duration(math.MaxInt64),
			want: &Duration{Seconds: maxGoSeconds, Nanos: int32(math.MaxInt64 - 1e9*maxGoSeconds)},
		},
	} {
		d := New(tt.in)

		if !reflect.DeepEqual(tt.want, d) {
			t.Errorf("unexpected duration: %v [want: %v]", d, tt.want)
		}
	}
}

func TestToDuration(t *testing.T) {
	for _, tt := range []struct {
		in   *Duration
		want time.Duration
	}{
		{want: time.Duration(0), in: &Duration{}},
		{want: -time.Second, in: &Duration{Seconds: -1}},
		{want: time.Second, in: &Duration{Seconds: 1}},
		{
			want: -time.Second - time.Millisecond,
			in:   &Duration{Seconds: -1, Nanos: -1e6},
		},
		{
			want: +time.Second + time.Millisecond,
			in:   &Duration{Seconds: 1, Nanos: 1e6},
		},
		{
			want: time.Duration(math.MinInt64),
			in:   &Duration{Seconds: minGoSeconds, Nanos: int32(math.MinInt64 - 1e9*minGoSeconds)},
		},
		{
			want: time.Duration(math.MaxInt64),
			in:   &Duration{Seconds: maxGoSeconds, Nanos: int32(math.MaxInt64 - 1e9*maxGoSeconds)},
		},
		{
			in:   &Duration{Seconds: math.MaxInt64},
			want: time.Duration(math.MaxInt64),
		},
		{
			in:   &Duration{Seconds: math.MinInt64},
			want: time.Duration(math.MinInt64),
		},
	} {
		d := tt.in.ToDuration()

		if tt.want != d {
			t.Errorf("unexpected duration: %v [want: %v]", d, tt.want)
		}
	}
}
