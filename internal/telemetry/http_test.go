package telemetry

import (
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
