// Package dealerships lists Ford dealerships and finds the ones nearest to a point.
package dealerships

import (
	"math"
	"sort"
	"sync"
)

type Dealership struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Phone   string  `json:"phone"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	// Rating is the average customer rating, 1-5.
	Rating   float64  `json:"rating"`
	Services []string `json:"services"`
	// Hours is Schedule in words, filled in by NewStore.
	Hours    string   `json:"hours"`
	Schedule Schedule `json:"-"`
	// DistanceKm is only set when results are sorted by proximity (a pointer so
	// a genuine 0 km is still sent).
	DistanceKm *float64 `json:"distanceKm,omitempty"`
}

// Store is the in-memory dealership directory. Offline it holds seed data;
// with Ford's platform it's a snapshot of Ford's dealer directory, swapped
// in by Replace on a refresh, so lookups never wait on the network.
type Store struct {
	mu    sync.RWMutex
	items []Dealership
}

func NewStore(seed []Dealership) *Store {
	s := &Store{}
	s.Replace(seed)
	return s
}

// Replace swaps in a new directory.
func (s *Store) Replace(items []Dealership) {
	for i := range items {
		items[i].Hours = items[i].Schedule.String()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = items
}

// ScheduleOf returns a dealership's opening schedule.
func (s *Store) ScheduleOf(id string) (Schedule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.items {
		if d.ID == id {
			return d.Schedule, true
		}
	}
	return Schedule{}, false
}

func (s *Store) All() []Dealership {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Dealership(nil), s.items...)
}

// Nearest returns every dealership sorted by distance from (lat, lng).
func (s *Store) Nearest(lat, lng float64) []Dealership {
	out := s.All()
	for i := range out {
		d := math.Round(haversineKm(lat, lng, out[i].Lat, out[i].Lng)*10) / 10
		out[i].DistanceKm = &d
	}
	sort.Slice(out, func(i, j int) bool { return *out[i].DistanceKm < *out[j].DistanceKm })
	return out
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLng := toRad(lat2-lat1), toRad(lng2-lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}
