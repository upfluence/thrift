package timestamp

import "time"

func Now() *Timestamp {
	return New(time.Now())
}

func New(t time.Time) *Timestamp {
	return &Timestamp{Seconds: int64(t.Unix()), Nanos: int32(t.Nanosecond())}
}

func (t *Timestamp) ToTime() time.Time {
	return time.Unix(t.Seconds, int64(t.Nanos)).UTC()
}
