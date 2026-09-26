package dealerships

import (
	"fmt"
	"time"
)

// OpenHours is one day's opening window in local time: whole hours, [Open, Close).
type OpenHours struct{ Open, Close int }

// Schedule is when a dealership takes service appointments.
type Schedule struct {
	Location *time.Location
	Weekdays OpenHours  // Monday to Friday
	Saturday *OpenHours // nil: closed. Sundays are always closed.
	Bays     int        // appointments per hourly slot
}

// On returns the opening hours for a weekday; ok is false when closed.
func (s Schedule) On(day time.Weekday) (OpenHours, bool) {
	switch day {
	case time.Sunday:
		return OpenHours{}, false
	case time.Saturday:
		if s.Saturday == nil {
			return OpenHours{}, false
		}
		return *s.Saturday, true
	default:
		return s.Weekdays, true
	}
}

// String is the human-readable form, e.g. "Mon–Fri 8:00–18:00 · Sat 8:00–12:00".
func (s Schedule) String() string {
	window := func(h OpenHours) string { return fmt.Sprintf("%d:00–%d:00", h.Open, h.Close) }
	switch {
	case s.Saturday == nil:
		return "Mon–Fri " + window(s.Weekdays)
	case *s.Saturday == s.Weekdays:
		return "Mon–Sat " + window(s.Weekdays)
	default:
		return "Mon–Fri " + window(s.Weekdays) + " · Sat " + window(*s.Saturday)
	}
}
