// Package hours decides whether a place is open, in Bangkok time.
package hours

import (
	"time"
)

// Bangkok is the timezone all opening hours are stored in.
var Bangkok = mustLoad("Asia/Bangkok")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Thailand has no DST, so a fixed offset is an exact fallback.
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}

// Interval is one opening window. Weekday 0 is Sunday. Opens and Closes are
// "HH:MM"; Closes <= Opens means the window runs past midnight.
type Interval struct {
	Weekday int    `json:"weekday"`
	Opens   string `json:"opens"`
	Closes  string `json:"closes"`
}

// Status reports whether any interval covers now, and when it closes.
func Status(intervals []Interval, now time.Time) (open bool, closesAt string) {
	now = now.In(Bangkok)
	today := int(now.Weekday())
	yesterday := (today + 6) % 7
	t := now.Hour()*60 + now.Minute()

	for _, iv := range intervals {
		o, okO := minutes(iv.Opens)
		c, okC := minutes(iv.Closes)
		if !okO || !okC {
			continue
		}
		overnight := c <= o
		switch {
		case iv.Weekday == today && !overnight && t >= o && t < c:
			return true, iv.Closes
		case iv.Weekday == today && overnight && t >= o:
			return true, iv.Closes
		case iv.Weekday == yesterday && overnight && t < c:
			return true, iv.Closes
		}
	}
	return false, ""
}

func minutes(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
