package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewStore([]Snapshot{{VIN: "VIN1", MileageKm: 1000}})).Register(mux)

	for path, want := range map[string]int{
		"/api/vehicles/VIN1/telemetry":    http.StatusOK,
		"/api/vehicles/UNKNOWN/telemetry": http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("%s: want %d, got %d", path, want, rec.Code)
		}
	}
}

type failing struct{}

func (failing) Get(context.Context, string) (Snapshot, bool, error) {
	return Snapshot{}, false, errors.New("ford down")
}

func TestUpstreamFailure(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(failing{}).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/vehicles/VIN1/telemetry", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d", rec.Code)
	}
}
