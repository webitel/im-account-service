package model

import "time"

// Clock interface
type Clock interface {
	// Now returns the current local time.
	Now() time.Time
}

// ClockFunc implements Clock interface
type ClockFunc func() time.Time

func (fn ClockFunc) Now() time.Time {
	return fn()
}

// LocalTime Clock
var LocalTime Clock = ClockFunc(time.Now)
