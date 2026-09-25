// Package dealerships lists Ford dealerships and finds the ones nearest to a point.
package dealerships

import (
	"math"
	"sort"
)

type Dealership struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Phone   string  `json:"phone"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	// DistanceKm is only set when results are sorted by proximity.
	DistanceKm float64 `json:"distanceKm,omitempty"`
}

// Store is a read-only in-memory dealership directory.
type Store struct{ items []Dealership }

func NewStore(seed []Dealership) *Store { return &Store{items: seed} }

func (s *Store) All() []Dealership {
	return append([]Dealership(nil), s.items...)
}

func (s *Store) Exists(id string) bool {
	for _, d := range s.items {
		if d.ID == id {
			return true
		}
	}
	return false
}

// Nearest returns every dealership sorted by distance from (lat, lng).
func (s *Store) Nearest(lat, lng float64) []Dealership {
	out := s.All()
	for i := range out {
		out[i].DistanceKm = math.Round(haversineKm(lat, lng, out[i].Lat, out[i].Lng)*10) / 10
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })
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
