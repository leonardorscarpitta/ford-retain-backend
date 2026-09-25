// Package notifications exposes the maintenance alerts sent to a vehicle owner
// (e.g. over WhatsApp) so they can be reviewed in the platform.
package notifications

import (
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

// Store is an in-memory notification repository.
type Store struct {
	mu    sync.RWMutex
	items []Notification
}

func NewStore(seed []Notification) *Store {
	return &Store{items: seed}
}

func (s *Store) ListByVIN(vin string) []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Notification{}
	for _, n := range s.items {
		if n.VIN == vin {
			out = append(out, n)
		}
	}
	return out
}

// MarkRead flags a notification as read. It reports whether it was found.
func (s *Store) MarkRead(id string) (Notification, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Read = true
			return s.items[i], true
		}
	}
	return Notification{}, false
}
