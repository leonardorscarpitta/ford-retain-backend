package appointments

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type fakeDealerships map[string]bool

func (f fakeDealerships) Exists(id string) bool { return f[id] }

func TestCreate(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	svc := NewService(fakeDealerships{"d1": true}, nil)
	svc.now = func() time.Time { return now }

	tests := []struct {
		name    string
		in      CreateInput
		wantErr bool
	}{
		{"valid", CreateInput{DealershipID: "d1", ScheduledAt: now.Add(24 * time.Hour), Service: "Oil change"}, false},
		{"missing service", CreateInput{DealershipID: "d1", ScheduledAt: now.Add(time.Hour)}, true},
		{"in the past", CreateInput{DealershipID: "d1", ScheduledAt: now.Add(-time.Hour), Service: "Brakes"}, true},
		{"unknown dealership", CreateInput{DealershipID: "nope", ScheduledAt: now.Add(time.Hour), Service: "Brakes"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create("VIN1", tt.in)
			if tt.wantErr != errors.Is(err, ErrInvalid) {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}

	if got := len(svc.ListByVIN("VIN1", WhenAll)); got != 1 {
		t.Errorf("want 1 stored appointment, got %d", got)
	}
}

func TestListByVINFiltersAndSorts(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	at := func(days int) time.Time { return now.Add(time.Duration(days) * 24 * time.Hour) }
	svc := NewService(fakeDealerships{}, []Appointment{
		{ID: "past-old", VIN: "VIN1", ScheduledAt: at(-30)},
		{ID: "future-late", VIN: "VIN1", ScheduledAt: at(20)},
		{ID: "past-recent", VIN: "VIN1", ScheduledAt: at(-2)},
		{ID: "future-soon", VIN: "VIN1", ScheduledAt: at(3)},
		{ID: "other-vin", VIN: "VIN2", ScheduledAt: at(-1)},
	})
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
			got := svc.ListByVIN("VIN1", tt.when)
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
