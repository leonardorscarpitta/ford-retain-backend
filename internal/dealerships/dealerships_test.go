package dealerships

import "testing"

func TestNearestSortsByDistance(t *testing.T) {
	s := NewStore([]Dealership{
		{ID: "far", Lat: -22.90, Lng: -43.17},  // Rio de Janeiro
		{ID: "near", Lat: -23.55, Lng: -46.63}, // São Paulo
	})

	got := s.Nearest(-23.56, -46.65)

	if got[0].ID != "near" || got[1].ID != "far" {
		t.Fatalf("want [near far], got [%s %s]", got[0].ID, got[1].ID)
	}
	if d := got[0].DistanceKm; d == nil || *d <= 0 || *d > 5 {
		t.Errorf("unexpected distance to near dealership: %v", d)
	}
}

func TestNearestKeepsZeroDistance(t *testing.T) {
	got := NewStore([]Dealership{{ID: "here", Lat: -22.9, Lng: -47.06}}).Nearest(-22.9, -47.06)
	if got[0].DistanceKm == nil || *got[0].DistanceKm != 0 {
		t.Fatalf("want distance 0, got %v", got[0].DistanceKm)
	}
}
