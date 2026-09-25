package vehicles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewStore([]Vehicle{{VIN: "A", Model: "F-150"}, {VIN: "B", Model: "Ranger"}})).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/vehicles", nil))
	var list []Vehicle
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil || len(list) != 2 || list[0].VIN != "A" {
		t.Fatalf("list: want [A B] in order, got %v (err %v)", list, err)
	}

	for path, want := range map[string]int{
		"/api/vehicles/B":       http.StatusOK,
		"/api/vehicles/MISSING": http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Errorf("%s: want %d, got %d", path, want, rec.Code)
		}
	}
}
