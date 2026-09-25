package vehicles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ford-retain/backend/internal/platform/identity"
)

func TestHTTP(t *testing.T) {
	h := NewHandler(NewStore([]Vehicle{
		{VIN: "A", OwnerID: "ana", Model: "F-150"},
		{VIN: "B", OwnerID: "bruno", Model: "Ranger"},
		{VIN: "C", OwnerID: "ana", Model: "Lightning"},
	}))
	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("/api/vehicles/{vin}/", h.RequireOwner(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	as := func(user, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		req = req.WithContext(identity.WithUserID(req.Context(), user))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	var list []Vehicle
	if err := json.NewDecoder(as("ana", "/api/vehicles").Body).Decode(&list); err != nil || len(list) != 2 || list[0].VIN != "A" || list[1].VIN != "C" {
		t.Fatalf("ana's garage: want [A C], got %v (err %v)", list, err)
	}

	tests := []struct {
		user, path string
		want       int
	}{
		{"ana", "/api/vehicles/A", http.StatusOK},
		{"ana", "/api/vehicles/B", http.StatusNotFound},           // someone else's
		{"ana", "/api/vehicles/A/telemetry", http.StatusTeapot},   // guard lets owner through
		{"ana", "/api/vehicles/B/telemetry", http.StatusNotFound}, // guard blocks others
		{"", "/api/vehicles/A/telemetry", http.StatusNotFound},    // no identity at all
		{"bruno", "/api/vehicles/MISSING/telemetry", http.StatusNotFound},
	}
	for _, tt := range tests {
		if rec := as(tt.user, tt.path); rec.Code != tt.want {
			t.Errorf("%s %s: want %d, got %d", tt.user, tt.path, tt.want, rec.Code)
		}
	}
}

type failing struct{}

func (failing) ForOwner(context.Context, string) ([]Vehicle, error) {
	return nil, errors.New("ford down")
}

func TestUpstreamFailure(t *testing.T) {
	h := NewHandler(failing{})
	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("/api/vehicles/{vin}/", h.RequireOwner(http.NotFoundHandler()))
	for _, path := range []string{"/api/vehicles", "/api/vehicles/A", "/api/vehicles/A/telemetry"} {
		req := httptest.NewRequest("GET", path, nil)
		req = req.WithContext(identity.WithUserID(req.Context(), "ana"))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadGateway {
			t.Errorf("%s: want 502, got %d", path, rec.Code)
		}
	}
}
