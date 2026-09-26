package appointments

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTP(t *testing.T) {
	now := time.Now()
	svc := NewService(func(id string, day time.Weekday) (Hours, bool) {
		if id != "d1" {
			return Hours{}, false
		}
		return alwaysOpen(id, day)
	}, NewMemoryRepo([]Appointment{
		{ID: "a1", VIN: "VIN1", DealershipID: "d1", ScheduledAt: now.Add(48 * time.Hour), Service: "Brakes"},
	}))
	mux := http.NewServeMux()
	h := NewHandler(svc)
	h.Register(mux)
	h.RegisterAvailability(mux)

	future := now.Add(72 * time.Hour).Truncate(time.Hour).UTC().Format(time.RFC3339)
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"list", "GET", "/api/vehicles/VIN1/appointments?when=upcoming", "", http.StatusOK},
		{"bad filter", "GET", "/api/vehicles/VIN1/appointments?when=soon", "", http.StatusBadRequest},
		{"create", "POST", "/api/vehicles/VIN1/appointments", `{"dealershipId":"d1","scheduledAt":"` + future + `","service":"Oil"}`, http.StatusCreated},
		{"create invalid", "POST", "/api/vehicles/VIN1/appointments", `{"dealershipId":"nope","scheduledAt":"` + future + `","service":"Oil"}`, http.StatusUnprocessableEntity},
		{"create bad json", "POST", "/api/vehicles/VIN1/appointments", `{"unknown":1}`, http.StatusBadRequest},
		{"availability", "GET", "/api/dealerships/d1/availability?days=3", "", http.StatusOK},
		{"availability from date", "GET", "/api/dealerships/d1/availability?from=2027-03-01", "", http.StatusOK},
		{"availability bad days", "GET", "/api/dealerships/d1/availability?days=90", "", http.StatusBadRequest},
		{"availability bad date", "GET", "/api/dealerships/d1/availability?from=tomorrow", "", http.StatusBadRequest},
		{"availability unknown", "GET", "/api/dealerships/zz/availability", "", http.StatusNotFound},
		{"cancel", "DELETE", "/api/vehicles/VIN1/appointments/a1", "", http.StatusNoContent},
		{"cancel again", "DELETE", "/api/vehicles/VIN1/appointments/a1", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, rec.Code, rec.Body)
			}
		})
	}
}
