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
	svc := NewService(fakeDealerships{"d1": true}, []Appointment{
		{ID: "a1", VIN: "VIN1", DealershipID: "d1", ScheduledAt: now.Add(48 * time.Hour), Service: "Brakes"},
	})
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	future := now.Add(72 * time.Hour).UTC().Format(time.RFC3339)
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"list", "GET", "/api/vehicles/VIN1/appointments?when=upcoming", "", http.StatusOK},
		{"bad filter", "GET", "/api/vehicles/VIN1/appointments?when=soon", "", http.StatusBadRequest},
		{"create", "POST", "/api/vehicles/VIN1/appointments", `{"dealershipId":"d1","scheduledAt":"` + future + `","service":"Oil"}`, http.StatusCreated},
		{"create invalid", "POST", "/api/vehicles/VIN1/appointments", `{"dealershipId":"nope","scheduledAt":"` + future + `","service":"Oil"}`, http.StatusUnprocessableEntity},
		{"create bad json", "POST", "/api/vehicles/VIN1/appointments", `{"unknown":1}`, http.StatusBadRequest},
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
