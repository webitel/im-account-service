package model

import (
	"time"
)

// Timestamp (epochtime) default spec.
var Timestamp = NewEpochtime(
	TimePrecision(time.Millisecond),
)

// type Timestamp int64

// var unixEpoch = time.Date(1970, time.January, 01, 00, 00, 00, 000000000, time.UTC)
var unixEpoch = time.Unix(0, 0).UTC()

// // Timestamp (epochtime) default precision
// const UnixToTimestamp = time.Millisecond

// Epochtime specification
type Epochtime struct {
	since time.Time
	prec  time.Duration
	tloc  *time.Location
}

func (spec *Epochtime) TimeEpoch() (since time.Time) {
	since = unixEpoch
	if spec != nil && since.Before(spec.since) {
		since = spec.since
	}
	return // since
}

func (spec *Epochtime) Precision() (prec time.Duration) {
	prec = time.Second
	if spec != nil && spec.prec > 0 {
		prec, _ = precision(spec.prec)
	}
	return // prec
}

func (spec *Epochtime) Location() (loc *time.Location) {
	loc = time.Local
	if spec != nil && spec.tloc != nil {
		loc = spec.tloc
	}
	return // loc
}

type EpochtimeOption func(spec *Epochtime)

func TimeEpoch(since time.Time) EpochtimeOption {
	return func(spec *Epochtime) {
		spec.since = since
	}
}

func TimePrecision(prec time.Duration) EpochtimeOption {
	return func(spec *Epochtime) {
		spec.prec = prec
	}
}

func TimeLocation(loc *time.Location) EpochtimeOption {
	return func(spec *Epochtime) {
		spec.tloc = loc
	}
}

func precision(as time.Duration) (time.Duration, bool) {
	ok := true
	switch as {
	case time.Second:
	case time.Millisecond:
	case time.Microsecond:
	case time.Nanosecond:
	default:
		as, ok = time.Second, false
	}
	return as, ok
}

func NewEpochtime(opts ...EpochtimeOption) (spec Epochtime) {
	for _, option := range opts {
		option(&spec)
	}
	return
}

// Time converts given [time.Time] to [epochtime]. Timestamp[precision]
func (spec *Epochtime) Time(date time.Time) (tsec int64) {
	if date.IsZero() || date.Before(spec.TimeEpoch()) {
		return 0
	}
	switch spec.Precision() {
	case time.Second, 0: // default: timestamp
		return date.Unix() // seconds
	case time.Millisecond:
		return date.UnixMilli() // milliseconds
	case time.Microsecond:
		return date.UnixMicro() // microseconds
	case time.Nanosecond:
		return date.UnixNano() // nanoseconds
		// default:
		// 	panic(fmt.Errorf("epochtime: invalid precision %s", spec.Precision))
	}
	return 0
}

// Date converts given [tsec] epochtime to [time.Time] presentation value.
func (spec *Epochtime) Date(tsec int64) (date time.Time) {
	if tsec > 0 {
		precision := spec.Precision()
		epochtimeToUnix := (int64)(time.Second / precision)                             // time.Second(1e9) / time.Millicesond(1e6) = 1e3
		date = time.Unix(tsec/epochtimeToUnix, tsec%epochtimeToUnix*(int64)(precision)) // *1e9) // *time.Second
		date = date.In(spec.Location())
	}
	return // date?
}
