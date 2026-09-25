// Package telemetry exposes vehicle health data: component condition and
// estimated battery life.
package telemetry

import (
	"context"
	"sync"
	"time"
)

type Status string

const (
	StatusGood      Status = "good"
	StatusAttention Status = "attention"
	StatusCritical  Status = "critical"
)

// Zone is where a component sits on the vehicle, used to place it on a diagram.
type Zone string

const (
	ZoneEngineBay  Zone = "engine-bay"
	ZoneFrontWheel Zone = "front-wheel"
	ZoneRearWheel  Zone = "rear-wheel"
	ZoneUnderbody  Zone = "underbody"
	ZoneCabin      Zone = "cabin"
)

type Component struct {
	Name   string `json:"name"`
	Zone   Zone   `json:"zone"`
	Health int    `json:"health"` // 0-100
	Status Status `json:"status"`
}

type Battery struct {
	HealthPercent       int `json:"healthPercent"`
	EstimatedMonthsLeft int `json:"estimatedMonthsLeft"`
	// History holds monthly health readings, oldest first.
	History []int `json:"history"`
}

type Snapshot struct {
	VIN        string      `json:"vin"`
	MileageKm  int         `json:"mileageKm"`
	Battery    Battery     `json:"battery"`
	Components []Component `json:"components"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// Source is where telemetry comes from: Store (seed data) offline, Ford's
// connected-vehicle API in production.
type Source interface {
	Get(ctx context.Context, vin string) (Snapshot, bool, error)
}

// Store is an in-memory telemetry repository keyed by VIN.
type Store struct {
	mu    sync.RWMutex
	byVIN map[string]Snapshot
}

func NewStore(seed []Snapshot) *Store {
	m := make(map[string]Snapshot, len(seed))
	for _, s := range seed {
		m[s.VIN] = s
	}
	return &Store{byVIN: m}
}

func (s *Store) Get(_ context.Context, vin string) (Snapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.byVIN[vin]
	return snap, ok, nil
}
