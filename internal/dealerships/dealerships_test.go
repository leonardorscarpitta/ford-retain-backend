package dealerships

import (
	"testing"
	"time"
)

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

func TestSchedule(t *testing.T) {
	sat := OpenHours{8, 12}
	s := Schedule{Weekdays: OpenHours{8, 18}, Saturday: &sat}
	if got := s.String(); got != "Mon–Fri 8:00–18:00 · Sat 8:00–12:00" {
		t.Errorf("String: %q", got)
	}
	same := OpenHours{7, 20}
	if got := (Schedule{Weekdays: same, Saturday: &same}).String(); got != "Mon–Sat 7:00–20:00" {
		t.Errorf("String (same Saturday): %q", got)
	}
	if _, open := s.On(time.Sunday); open {
		t.Error("Sundays must be closed")
	}
	if h, open := s.On(time.Saturday); !open || h != sat {
		t.Errorf("Saturday: %v %v", h, open)
	}
}
