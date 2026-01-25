package model

import (
	"testing"
	"time"
)

func TestUnixEpoch(test *testing.T) {
	if !time.Unix(0, 0).UTC().Equal(time.Date(1970, time.January, 01, 00, 00, 00, 000000000, time.UTC)) {
		test.Errorf("UnixEpoch: ambiguous")
	}
}

func TestEpochtime(test *testing.T) {

	epoch := NewEpochtime(
		TimePrecision(time.Millisecond),
	)

	date := LocalTime.Now().Round(epoch.Precision())
	tsec := epoch.Time(date)

	if want := date.UnixMilli(); tsec != want {
		test.Errorf("epoch.Time(date) = %d, want %d", tsec, want)
		return
	}

	if got := epoch.Date(tsec); !got.Equal(date) {
		test.Errorf("epoch.Date(tsec) = %q, want %q", got, date)
		return
	}
}
