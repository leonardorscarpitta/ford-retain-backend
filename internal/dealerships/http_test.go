package dealerships

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewStore([]Dealership{{ID: "d1", Lat: -23.5, Lng: -46.6}})).Register(mux)

	for path, want := range map[string]int{
		"/api/dealerships":                   http.StatusOK,
		"/api/dealerships?lat=-23.5&lng=-46": http.StatusOK,
		"/api/dealerships?lat=abc&lng=-46":   http.StatusBadRequest,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("%s: want %d, got %d", path, want, rec.Code)
		}
	}
}
