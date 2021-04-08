package duration

import (
	"math"
	"time"
)

func New(d time.Duration) *Duration {
	var nanos = d.Nanoseconds()

	secs := nanos / 1e9
	nanos -= secs * 1e9

	return &Duration{Seconds: int64(secs), Nanos: int32(nanos)}
}

func (d *Duration) ToDuration() time.Duration {
	var (
		secs     = d.Seconds
		nanos    = d.Nanos
		res      = time.Duration(secs) * time.Second
		overflow = res/time.Second != time.Duration(secs)
	)

	res += time.Duration(nanos) * time.Nanosecond
	overflow = overflow || (secs < 0 && nanos < 0 && res > 0)
	overflow = overflow || (secs > 0 && nanos > 0 && res < 0)

	if overflow {
		switch {
		case secs < 0:
			return time.Duration(math.MinInt64)
		case secs > 0:
			return time.Duration(math.MaxInt64)
		}
	}

	return res
}
