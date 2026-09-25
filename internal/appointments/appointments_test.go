package appointments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var ctx = context.Background()

// brt is a fixed UTC−3 zone, like São Paulo.
var brt = time.FixedZone("BRT", -3*60*60)

// weekdayHours: dealership "d1" is open Mon–Fri 8:00–18:00 with one bay;
// every other ID is unknown.
func weekdayHours(id string, day time.Weekday) (Hours, bool) {
	if id != "d1" {
		return Hours{}, false
	}
	if day == time.Saturday || day == time.Sunday {
		return Hours{Location: brt}, true
	}
	return Hours{Open: 8, Close: 18, Bays: 1, Location: brt}, true
}

// alwaysOpen is for tests that don't care about opening hours.
func alwaysOpen(string, time.Weekday) (Hours, bool) {
	return Hours{Open: 0, Close: 24, Bays: 100, Location: time.UTC}, true
}

func TestCreate(t *testing.T) {
	now := time.Date(2026, 1, 5, 9, 30, 0, 0, brt) // Monday 09:30 local
	svc := NewService(weekdayHours, NewMemoryRepo(nil))
	svc.now = func() time.Time { return now }
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 1, day, hour, minute, 0, 0, brt) }

	tests := []struct {
		name    string
		in      CreateInput
		wantErr string // "" for success
	}{
		{"open slot", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 10, 0), Service: "Oil change"}, ""},
		{"same slot again (1 bay)", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 10, 0), Service: "Brakes"}, "fully booked"},
		{"same instant in UTC", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 10, 0).UTC(), Service: "Brakes"}, "fully booked"},
		{"missing service", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 11, 0)}, "service is required"},
		{"in the past", CreateInput{DealershipID: "d1", ScheduledAt: at(5, 9, 0), Service: "Brakes"}, "future"},
		{"unknown dealership", CreateInput{DealershipID: "nope", ScheduledAt: at(6, 11, 0), Service: "Brakes"}, "unknown dealership"},
		{"not on the hour", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 11, 30), Service: "Brakes"}, "available times"},
		{"after closing", CreateInput{DealershipID: "d1", ScheduledAt: at(6, 18, 0), Service: "Brakes"}, "available times"},
		{"closed on Sunday", CreateInput{DealershipID: "d1", ScheduledAt: at(11, 10, 0), Service: "Brakes"}, "available times"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, "VIN1", tt.in)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want ErrInvalid containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAvailability(t *testing.T) {
	now := time.Date(2026, 1, 5, 9, 30, 0, 0, brt) // Monday 09:30 local
	svc := NewService(weekdayHours, NewMemoryRepo([]Appointment{
		{ID: "a", DealershipID: "d1", ScheduledAt: time.Date(2026, 1, 6, 10, 0, 0, 0, brt)},
	}))
	svc.now = func() time.Time { return now }

	days, loc, err := svc.Availability(ctx, "d1", now, 7)
	if err != nil || loc != brt || len(days) != 7 {
		t.Fatalf("got %d days, loc %v, err %v", len(days), loc, err)
	}
	// Monday: slots before 09:30 are gone; 10:00–17:00 remain.
	if first := days[0].Slots[0].Start; first.Hour() != 10 || len(days[0].Slots) != 8 {
		t.Errorf("today: first slot %v, %d slots", first, len(days[0].Slots))
	}
	// Tuesday: 10 slots, the booked 10:00 one is full.
	for _, slot := range days[1].Slots {
		if want := slot.Start.Hour() != 10; slot.Available != want {
			t.Errorf("Tuesday %v: available=%v", slot.Start, slot.Available)
		}
	}
	// Saturday and Sunday are closed.
	if days[5].Date != "2026-01-10" || len(days[5].Slots) != 0 || len(days[6].Slots) != 0 {
		t.Errorf("weekend should have no slots: %v %v", days[5], days[6])
	}
	if _, _, err := svc.Availability(ctx, "nope", now, 7); !errors.Is(err, ErrUnknownDealership) {
		t.Errorf("unknown dealership: %v", err)
	}
}

func TestListByVINFiltersAndSorts(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	at := func(days int) time.Time { return now.Add(time.Duration(days) * 24 * time.Hour) }
	svc := NewService(alwaysOpen, NewMemoryRepo([]Appointment{
		{ID: "past-old", VIN: "VIN1", ScheduledAt: at(-30)},
		{ID: "future-late", VIN: "VIN1", ScheduledAt: at(20)},
		{ID: "past-recent", VIN: "VIN1", ScheduledAt: at(-2)},
		{ID: "future-soon", VIN: "VIN1", ScheduledAt: at(3)},
		{ID: "other-vin", VIN: "VIN2", ScheduledAt: at(-1)},
	}))
	svc.now = func() time.Time { return now }

	tests := []struct {
		when When
		want []string
	}{
		{WhenPast, []string{"past-recent", "past-old"}},
		{WhenUpcoming, []string{"future-soon", "future-late"}},
		{WhenAll, []string{"past-old", "past-recent", "future-soon", "future-late"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.when), func(t *testing.T) {
			got, _ := svc.ListByVIN(ctx, "VIN1", tt.when)
			ids := make([]string, len(got))
			for i, a := range got {
				ids[i] = a.ID
			}
			if fmt.Sprint(ids) != fmt.Sprint(tt.want) {
				t.Errorf("want %v, got %v", tt.want, ids)
			}
		})
	}
}

func TestCancel(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	svc := NewService(alwaysOpen, NewMemoryRepo([]Appointment{
		{ID: "future", VIN: "VIN1", ScheduledAt: now.Add(time.Hour)},
		{ID: "past", VIN: "VIN1", ScheduledAt: now.Add(-time.Hour)},
	}))
	svc.now = func() time.Time { return now }

	if err := svc.Cancel(ctx, "VIN2", "future"); !errors.Is(err, ErrNotFound) {
		t.Errorf("other vehicle's appointment: want ErrNotFound, got %v", err)
	}
	if err := svc.Cancel(ctx, "VIN1", "past"); !errors.Is(err, ErrInvalid) {
		t.Errorf("past appointment: want ErrInvalid, got %v", err)
	}
	if err := svc.Cancel(ctx, "VIN1", "future"); err != nil {
		t.Fatalf("upcoming appointment: %v", err)
	}
	if got, _ := svc.ListByVIN(ctx, "VIN1", WhenUpcoming); len(got) != 0 {
		t.Errorf("want no upcoming appointments after cancel, got %d", len(got))
	}
}
