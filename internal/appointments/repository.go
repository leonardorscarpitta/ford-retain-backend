package appointments

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Repository stores appointments. MemoryRepo keeps them in process (tests,
// offline development); PostgresRepo persists them.
type Repository interface {
	// ListByVIN returns every appointment for a vehicle, in any order.
	ListByVIN(ctx context.Context, vin string) ([]Appointment, error)
	// Get returns one of a vehicle's appointments.
	Get(ctx context.Context, vin, id string) (Appointment, bool, error)
	// Book stores a (assigning its ID) unless `bays` appointments already
	// start at the same dealership and time; ok is false when the slot is
	// full. The check and the insert must be atomic.
	Book(ctx context.Context, a Appointment, bays int) (Appointment, bool, error)
	// Delete removes one of a vehicle's appointments.
	Delete(ctx context.Context, vin, id string) error
	// Booked counts appointments per start time at a dealership in [from, to).
	Booked(ctx context.Context, dealershipID string, from, to time.Time) (map[time.Time]int, error)
}

// MemoryRepo is an in-process Repository.
type MemoryRepo struct {
	mu     sync.RWMutex
	items  []Appointment
	nextID int
}

// NewMemoryRepo starts with the given appointments (stored as-is).
func NewMemoryRepo(seed []Appointment) *MemoryRepo {
	return &MemoryRepo{items: append([]Appointment(nil), seed...), nextID: 1}
}

func (m *MemoryRepo) ListByVIN(_ context.Context, vin string) ([]Appointment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Appointment{}
	for _, a := range m.items {
		if a.VIN == vin {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *MemoryRepo) Get(_ context.Context, vin, id string) (Appointment, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.items {
		if a.ID == id && a.VIN == vin {
			return a, true, nil
		}
	}
	return Appointment{}, false, nil
}

func (m *MemoryRepo) Book(_ context.Context, a Appointment, bays int) (Appointment, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	booked := 0
	for _, b := range m.items {
		if b.DealershipID == a.DealershipID && b.ScheduledAt.Equal(a.ScheduledAt) {
			booked++
		}
	}
	if booked >= bays {
		return Appointment{}, false, nil
	}
	a.ID = fmt.Sprintf("apt-%d", m.nextID)
	m.nextID++
	m.items = append(m.items, a)
	return a, true, nil
}

func (m *MemoryRepo) Delete(_ context.Context, vin, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, a := range m.items {
		if a.ID == id && a.VIN == vin {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MemoryRepo) Booked(_ context.Context, dealershipID string, from, to time.Time) (map[time.Time]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[time.Time]int{}
	for _, a := range m.items {
		if a.DealershipID == dealershipID && !a.ScheduledAt.Before(from) && a.ScheduledAt.Before(to) {
			out[a.ScheduledAt.UTC()]++
		}
	}
	return out, nil
}
