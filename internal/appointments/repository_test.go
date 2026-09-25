package appointments

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryRepo(t *testing.T) {
	testRepository(t, func(t *testing.T) Repository { return NewMemoryRepo(nil) })
}

// testRepository is the contract every Repository must meet.
func testRepository(t *testing.T, newRepo func(t *testing.T) Repository) {
	slot := time.Date(2030, 3, 4, 10, 0, 0, 0, time.UTC)
	appt := func(vin, dealer string, at time.Time) Appointment {
		return Appointment{VIN: vin, DealershipID: dealer, ScheduledAt: at, Service: "Oil", CreatedAt: slot.Add(-time.Hour)}
	}

	t.Run("book, list, get, delete", func(t *testing.T) {
		r := newRepo(t)
		a, ok, err := r.Book(ctx, appt("VIN1", "d1", slot), 2)
		if err != nil || !ok || a.ID == "" {
			t.Fatalf("book: %+v %v %v", a, ok, err)
		}
		r.Book(ctx, appt("VIN2", "d1", slot.Add(time.Hour)), 2)

		list, err := r.ListByVIN(ctx, "VIN1")
		if err != nil || len(list) != 1 || list[0].ID != a.ID || !list[0].ScheduledAt.Equal(slot) || list[0].Service != "Oil" {
			t.Fatalf("list: %+v %v", list, err)
		}
		if _, ok, _ := r.Get(ctx, "VIN2", a.ID); ok {
			t.Fatal("Get must be scoped to the vehicle")
		}
		if got, ok, err := r.Get(ctx, "VIN1", a.ID); err != nil || !ok || got.DealershipID != "d1" {
			t.Fatalf("get: %+v %v %v", got, ok, err)
		}
		if err := r.Delete(ctx, "VIN1", a.ID); err != nil {
			t.Fatal(err)
		}
		if list, _ := r.ListByVIN(ctx, "VIN1"); len(list) != 0 {
			t.Fatalf("after delete: %+v", list)
		}
	})

	t.Run("capacity per dealership and time", func(t *testing.T) {
		r := newRepo(t)
		for i, want := range []bool{true, true, false} {
			if _, ok, err := r.Book(ctx, appt("VIN1", "d1", slot), 2); err != nil || ok != want {
				t.Fatalf("booking %d: ok=%v err=%v", i+1, ok, err)
			}
		}
		if _, ok, _ := r.Book(ctx, appt("VIN1", "d2", slot), 2); !ok {
			t.Fatal("another dealership's slot is independent")
		}
		if _, ok, _ := r.Book(ctx, appt("VIN1", "d1", slot.Add(time.Hour)), 2); !ok {
			t.Fatal("another hour is independent")
		}
	})

	t.Run("booked counts in range", func(t *testing.T) {
		r := newRepo(t)
		r.Book(ctx, appt("VIN1", "d1", slot), 3)
		r.Book(ctx, appt("VIN2", "d1", slot.In(brt)), 3) // same instant, other zone
		r.Book(ctx, appt("VIN1", "d1", slot.Add(24*time.Hour)), 3)
		r.Book(ctx, appt("VIN1", "d9", slot), 3)
		got, err := r.Booked(ctx, "d1", slot, slot.Add(time.Hour))
		if err != nil || len(got) != 1 || got[slot] != 2 {
			t.Fatalf("booked: %v %v", got, err)
		}
	})

	t.Run("concurrent bookings never exceed bays", func(t *testing.T) {
		r := newRepo(t)
		var wg sync.WaitGroup
		var mu sync.Mutex
		won := 0
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, ok, err := r.Book(ctx, appt("VIN1", "d1", slot), 2)
				if err != nil {
					t.Error(err)
				}
				if ok {
					mu.Lock()
					won++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if won != 2 {
			t.Fatalf("want exactly 2 bookings for 2 bays, got %d", won)
		}
	})
}
