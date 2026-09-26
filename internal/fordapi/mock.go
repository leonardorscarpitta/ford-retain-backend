package fordapi

import (
	"crypto/subtle"
	"net/http"

	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

// MockData is what the mock Ford API serves.
type MockData struct {
	Vehicles  []vehicles.Vehicle
	Telemetry []telemetry.Snapshot
	Alerts    []notifications.Notification
	Dealers   []dealerships.Dealership
}

// NewMock returns a server implementing the contract in the package doc,
// for local development (cmd/fordmock) and tests. Requests must carry the
// given API key. Alert read state is kept in memory.
func NewMock(apiKey string, data MockData) http.Handler {
	vehicleStore := vehicles.NewStore(data.Vehicles)
	telemetryStore := telemetry.NewStore(data.Telemetry)
	alertStore := notifications.NewStore(data.Alerts)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/owners/{owner}/vehicles", func(w http.ResponseWriter, r *http.Request) {
		list, _ := vehicleStore.ForOwner(r.Context(), r.PathValue("owner"))
		httpx.JSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /v1/vehicles/{vin}/telemetry", func(w http.ResponseWriter, r *http.Request) {
		snap, ok, _ := telemetryStore.Get(r.Context(), r.PathValue("vin"))
		if !ok {
			httpx.Error(w, http.StatusNotFound, "no telemetry")
			return
		}
		httpx.JSON(w, http.StatusOK, snap)
	})
	mux.HandleFunc("GET /v1/vehicles/{vin}/alerts", func(w http.ResponseWriter, r *http.Request) {
		list, _ := alertStore.ListByVIN(r.Context(), r.PathValue("vin"))
		httpx.JSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("POST /v1/vehicles/{vin}/alerts/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		n, ok, _ := alertStore.MarkRead(r.Context(), r.PathValue("vin"), r.PathValue("id"))
		if !ok {
			httpx.Error(w, http.StatusNotFound, "no such alert")
			return
		}
		httpx.JSON(w, http.StatusOK, n)
	})
	mux.HandleFunc("GET /v1/dealers", func(w http.ResponseWriter, _ *http.Request) {
		out := make([]dealer, 0, len(data.Dealers))
		for _, d := range data.Dealers {
			wire := dealer{
				ID: d.ID, Name: d.Name, Address: d.Address, Phone: d.Phone, Lat: d.Lat, Lng: d.Lng,
				Rating: d.Rating, Services: d.Services, TimeZone: d.Schedule.Location.String(),
				Weekdays: hours(d.Schedule.Weekdays), Bays: d.Schedule.Bays,
			}
			if d.Schedule.Saturday != nil {
				sat := hours(*d.Schedule.Saturday)
				wire.Saturday = &sat
			}
			out = append(out, wire)
		}
		httpx.JSON(w, http.StatusOK, out)
	})

	want := []byte("Bearer " + apiKey)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			httpx.Error(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
