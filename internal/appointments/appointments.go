// Package appointments lets an owner schedule maintenance at a dealership.
package appointments

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Appointment struct {
	ID           string    `json:"id"`
	VIN          string    `json:"vin"`
	DealershipID string    `json:"dealershipId"`
	ScheduledAt  time.Time `json:"scheduledAt"`
	Service      string    `json:"service"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreateInput struct {
	DealershipID string    `json:"dealershipId"`
	ScheduledAt  time.Time `json:"scheduledAt"`
	Service      string    `json:"service"`
	Notes        string    `json:"notes"`
}

// Hours is a dealership's opening window on one day, in its local time:
// whole hours [Open, Close), and how many appointments fit in each hour.
type Hours struct {
	Open, Close int
	Bays        int
	Location    *time.Location
}

// Closed reports whether the dealership takes no appointments that day.
func (h Hours) Closed() bool { return h.Open >= h.Close || h.Bays < 1 }

// HoursLookup is what this module needs from the dealerships module: the
// opening hours of a dealership on a weekday (ok is false if it doesn't
// exist). A function keeps the modules decoupled; routes.go adapts it.
type HoursLookup func(dealershipID string, day time.Weekday) (h Hours, ok bool)

var (
	ErrInvalid           = errors.New("invalid appointment")
	ErrNotFound          = errors.New("appointment not found")
	ErrUnknownDealership = errors.New("unknown dealership")
)

// When filters appointments by their scheduled time relative to now.
type When string

const (
	WhenAll      When = ""
	WhenUpcoming When = "upcoming"
	WhenPast     When = "past"
)

func (w When) Valid() bool {
	return w == WhenAll || w == WhenUpcoming || w == WhenPast
}

type Service struct {
	hours HoursLookup
	now   func() time.Time

	mu     sync.RWMutex
	items  []Appointment
	nextID int
}

// NewService creates the service with optional seed appointments. Seeded
// appointments skip validation so past ones can be loaded.
func NewService(hours HoursLookup, seed []Appointment) *Service {
	return &Service{hours: hours, now: time.Now, items: seed, nextID: 1}
}

// ListByVIN returns a vehicle's appointments. Upcoming ones are sorted soonest
// first; past ones most recent first.
func (s *Service) ListByVIN(vin string, when When) []Appointment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := s.now()
	out := []Appointment{}
	for _, a := range s.items {
		if a.VIN != vin {
			continue
		}
		past := !a.ScheduledAt.After(now)
		if (when == WhenUpcoming && past) || (when == WhenPast && !past) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if when == WhenPast {
			return out[i].ScheduledAt.After(out[j].ScheduledAt)
		}
		return out[i].ScheduledAt.Before(out[j].ScheduledAt)
	})
	return out
}

// Create books a slot. The time must be the start of an open hourly slot at
// the dealership, in the future, with a bay still free.
func (s *Service) Create(vin string, in CreateInput) (Appointment, error) {
	if strings.TrimSpace(in.Service) == "" {
		return Appointment{}, fmt.Errorf("%w: service is required", ErrInvalid)
	}
	if !in.ScheduledAt.After(s.now()) {
		return Appointment{}, fmt.Errorf("%w: scheduledAt must be in the future", ErrInvalid)
	}
	loc, ok := s.location(in.DealershipID)
	if !ok {
		return Appointment{}, fmt.Errorf("%w: unknown dealership", ErrInvalid)
	}
	local := in.ScheduledAt.In(loc)
	h, _ := s.hours(in.DealershipID, local.Weekday())
	if h.Closed() || local.Minute() != 0 || local.Second() != 0 || local.Nanosecond() != 0 ||
		local.Hour() < h.Open || local.Hour() >= h.Close {
		return Appointment{}, fmt.Errorf("%w: pick one of the dealership's available times", ErrInvalid)
	}

	// Check capacity and insert under one lock so two bookings can't both
	// take the last bay.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bookedLocked(in.DealershipID, in.ScheduledAt) >= h.Bays {
		return Appointment{}, fmt.Errorf("%w: that time is fully booked", ErrInvalid)
	}
	a := Appointment{
		ID:           fmt.Sprintf("apt-%d", s.nextID),
		VIN:          vin,
		DealershipID: in.DealershipID,
		ScheduledAt:  in.ScheduledAt,
		Service:      strings.TrimSpace(in.Service),
		Notes:        in.Notes,
		CreatedAt:    s.now(),
	}
	s.nextID++
	s.items = append(s.items, a)
	return a, nil
}

// Cancel removes an upcoming appointment. Past appointments are history and
// can't be cancelled.
func (s *Service) Cancel(vin, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.items {
		if a.ID != id || a.VIN != vin {
			continue
		}
		if !a.ScheduledAt.After(s.now()) {
			return fmt.Errorf("%w: past appointments can't be cancelled", ErrInvalid)
		}
		s.items = append(s.items[:i], s.items[i+1:]...)
		return nil
	}
	return ErrNotFound
}

// Slot is one bookable hour at a dealership.
type Slot struct {
	Start     time.Time `json:"start"`
	Available bool      `json:"available"`
}

// Day lists a date's slots (none when the dealership is closed).
type Day struct {
	Date  string `json:"date"` // YYYY-MM-DD in the dealership's time zone
	Slots []Slot `json:"slots"`
}

// Availability lists future hourly slots for `days` days starting on the
// calendar date of `from` in the dealership's time zone. A slot is available
// while fewer appointments than bays are booked in it.
func (s *Service) Availability(dealershipID string, from time.Time, days int) ([]Day, *time.Location, error) {
	loc, ok := s.location(dealershipID)
	if !ok {
		return nil, nil, ErrUnknownDealership
	}
	now := s.now()
	y, m, d := from.In(loc).Date()

	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Day, 0, days)
	for i := 0; i < days; i++ {
		date := time.Date(y, m, d+i, 0, 0, 0, 0, loc)
		day := Day{Date: date.Format(time.DateOnly), Slots: []Slot{}}
		if h, _ := s.hours(dealershipID, date.Weekday()); !h.Closed() {
			for hour := h.Open; hour < h.Close; hour++ {
				start := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, loc)
				if !start.After(now) {
					continue
				}
				day.Slots = append(day.Slots, Slot{Start: start, Available: s.bookedLocked(dealershipID, start) < h.Bays})
			}
		}
		out = append(out, day)
	}
	return out, loc, nil
}

// location returns the dealership's time zone, or false if it doesn't exist.
func (s *Service) location(dealershipID string) (*time.Location, bool) {
	h, ok := s.hours(dealershipID, time.Monday)
	if !ok || h.Location == nil {
		return nil, false
	}
	return h.Location, true
}

// bookedLocked counts appointments at a dealership starting at t. Callers hold s.mu.
func (s *Service) bookedLocked(dealershipID string, t time.Time) int {
	n := 0
	for _, a := range s.items {
		if a.DealershipID == dealershipID && a.ScheduledAt.Equal(t) {
			n++
		}
	}
	return n
}
