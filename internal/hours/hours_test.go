package hours

import (
	"testing"
	"time"
)

func at(weekday time.Weekday, hh, mm int) time.Time {
	// 2026-10-04 is a Sunday.
	return time.Date(2026, 10, 4+int(weekday), hh, mm, 0, 0, Bangkok)
}

func TestStatus(t *testing.T) {
	daytime := []Interval{{Weekday: 1, Opens: "09:00", Closes: "18:00"}}
	overnight := []Interval{{Weekday: 5, Opens: "16:00", Closes: "00:00"}, {Weekday: 6, Opens: "16:00", Closes: "02:00"}}

	cases := []struct {
		name      string
		hours     []Interval
		now       time.Time
		wantOpen  bool
		wantClose string
	}{
		{"before opening", daytime, at(time.Monday, 8, 59), false, ""},
		{"at opening", daytime, at(time.Monday, 9, 0), true, "18:00"},
		{"at closing", daytime, at(time.Monday, 18, 0), false, ""},
		{"wrong day", daytime, at(time.Tuesday, 12, 0), false, ""},
		{"overnight evening", overnight, at(time.Friday, 23, 30), true, "00:00"},
		{"closes at midnight", overnight, at(time.Saturday, 0, 0), false, ""},
		{"after midnight from saturday", overnight, at(time.Sunday, 1, 30), true, "02:00"},
		{"after overnight close", overnight, at(time.Sunday, 2, 0), false, ""},
		{"utc input converted", daytime, at(time.Monday, 10, 0).UTC(), true, "18:00"},
	}
	for _, c := range cases {
		open, closes := Status(c.hours, c.now)
		if open != c.wantOpen || closes != c.wantClose {
			t.Errorf("%s: got (%v, %q), want (%v, %q)", c.name, open, closes, c.wantOpen, c.wantClose)
		}
	}
}
