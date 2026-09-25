package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP(t *testing.T) {
	store := NewStore([]Notification{
		{ID: "n1", VIN: "VIN1", Title: "Brakes"},
		{ID: "n2", VIN: "VIN2", Title: "Oil"},
	})
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/vehicles/VIN1/notifications", nil))
	var list []Notification
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil || len(list) != 1 || list[0].ID != "n1" {
		t.Fatalf("list: want only n1, got %v (err %v)", list, err)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("PATCH", "/api/notifications/n1/read", nil))
	if rec.Code != http.StatusOK || !store.ListByVIN("VIN1")[0].Read {
		t.Fatalf("mark read: status %d, read=%v", rec.Code, store.ListByVIN("VIN1")[0].Read)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("PATCH", "/api/notifications/missing/read", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing: want 404, got %d", rec.Code)
	}
}
