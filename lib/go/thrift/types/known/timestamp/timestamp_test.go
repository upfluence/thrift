package timestamp

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestTimestamp(t *testing.T) {
	for _, tt := range []struct {
		in   time.Time
		want *Timestamp
	}{
		{
			in:   time.Time{},
			want: &Timestamp{Seconds: -62135596800, Nanos: 0},
		},
		{
			in:   time.Unix(0, 0),
			want: &Timestamp{Seconds: 0, Nanos: 0},
		},
		{
			in:   time.Unix(math.MinInt64, 0),
			want: &Timestamp{Seconds: math.MinInt64, Nanos: 0},
		},
		{
			in:   time.Unix(math.MaxInt64, 1e9-1),
			want: &Timestamp{Seconds: math.MaxInt64, Nanos: 1e9 - 1},
		},
		{
			in:   time.Date(2021, 4, 5, 3, 25, 45, 940483, time.UTC),
			want: &Timestamp{Seconds: 1617593145, Nanos: 940483},
		},
	} {
		res := New(tt.in)

		if !reflect.DeepEqual(tt.want, res) {
			t.Errorf("unexpected timestamp: %v [want: %v]", res, tt.want)
		}
	}
}

func TestToTime(t *testing.T) {
	for _, tt := range []struct {
		in   *Timestamp
		want time.Time
	}{
		{
			in:   &Timestamp{Seconds: -62135596800, Nanos: 0},
			want: time.Time{},
		},
		{
			in:   &Timestamp{Seconds: 0, Nanos: 0},
			want: time.Unix(0, 0),
		},
		{
			in:   &Timestamp{Seconds: math.MinInt64, Nanos: 0},
			want: time.Unix(math.MinInt64, 0),
		},
		{
			in:   &Timestamp{Seconds: math.MaxInt64, Nanos: 1e9 - 1},
			want: time.Unix(math.MaxInt64, 1e9-1),
		},
		{
			in:   &Timestamp{Seconds: 1617593145, Nanos: 940483},
			want: time.Date(2021, 4, 5, 3, 25, 45, 940483, time.UTC),
		},
	} {
		if res := tt.in.ToTime(); !res.Equal(tt.want) {
			t.Errorf("unexpected timestamp: %v [want: %v]", res, tt.want)
		}
	}
}
