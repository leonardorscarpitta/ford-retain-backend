// Package notifications exposes the maintenance alerts sent to a vehicle owner
// (e.g. over WhatsApp) so they can be reviewed in the platform.
package notifications

import (
	"context"
	"sync"
	"time"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Notification struct {
	ID        string    `json:"id"`
	VIN       string    `json:"vin"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Severity  Severity  `json:"severity"`
	Channel   string    `json:"channel"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

// Source is where alerts come from: Store (seed data) offline, the WhatsApp
// alert history in Ford's platform in production.
type Source interface {
	ListByVIN(ctx context.Context, vin string) ([]Notification, error)
	MarkRead(ctx context.Context, vin, id string) (Notification, bool, error)
}

// Store is an in-memory notification repository.
type Store struct {
	mu    sync.RWMutex
	items []Notification
}

func NewStore(seed []Notification) *Store {
	return &Store{items: seed}
}

func (s *Store) ListByVIN(_ context.Context, vin string) ([]Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Notification{}
	for _, n := range s.items {
		if n.VIN == vin {
			out = append(out, n)
		}
	}
	return out, nil
}

// MarkRead flags one of a vehicle's notifications as read. It reports whether
// it was found.
func (s *Store) MarkRead(_ context.Context, vin, id string) (Notification, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id && s.items[i].VIN == vin {
			s.items[i].Read = true
			return s.items[i], true, nil
		}
	}
	return Notification{}, false, nil
}
