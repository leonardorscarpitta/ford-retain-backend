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

// DealershipChecker is what this module needs from the dealerships module.
// Depending on a small interface keeps modules decoupled.
type DealershipChecker interface {
	Exists(id string) bool
}

var (
	ErrInvalid  = errors.New("invalid appointment")
	ErrNotFound = errors.New("appointment not found")
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
	dealerships DealershipChecker
	now         func() time.Time

	mu     sync.RWMutex
	items  []Appointment
	nextID int
}

// NewService creates the service with optional seed appointments. Seeded
// appointments skip validation so past ones can be loaded.
func NewService(d DealershipChecker, seed []Appointment) *Service {
	return &Service{dealerships: d, now: time.Now, items: seed, nextID: 1}
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

func (s *Service) Create(vin string, in CreateInput) (Appointment, error) {
	switch {
	case strings.TrimSpace(in.Service) == "":
		return Appointment{}, fmt.Errorf("%w: service is required", ErrInvalid)
	case !in.ScheduledAt.After(s.now()):
		return Appointment{}, fmt.Errorf("%w: scheduledAt must be in the future", ErrInvalid)
	case !s.dealerships.Exists(in.DealershipID):
		return Appointment{}, fmt.Errorf("%w: unknown dealership", ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
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
