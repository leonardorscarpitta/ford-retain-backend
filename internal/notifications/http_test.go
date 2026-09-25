package notifications

import (
	"context"
	"encoding/json"
	"errors"
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
	mux.ServeHTTP(rec, httptest.NewRequest("PATCH", "/api/vehicles/VIN2/notifications/n1/read", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other vehicle's notification: want 404, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("PATCH", "/api/vehicles/VIN1/notifications/n1/read", nil))
	if rec.Code != http.StatusOK || !mustList(t, store)[0].Read {
		t.Fatalf("mark read: status %d, read=%v", rec.Code, mustList(t, store)[0].Read)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("PATCH", "/api/vehicles/VIN1/notifications/missing/read", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing: want 404, got %d", rec.Code)
	}
}

func mustList(t *testing.T, s *Store) []Notification {
	t.Helper()
	list, err := s.ListByVIN(context.Background(), "VIN1")
	if err != nil {
		t.Fatal(err)
	}
	return list
}

type failing struct{}

func (failing) ListByVIN(context.Context, string) ([]Notification, error) {
	return nil, errors.New("ford down")
}
func (failing) MarkRead(context.Context, string, string) (Notification, bool, error) {
	return Notification{}, false, errors.New("ford down")
}

func TestUpstreamFailure(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(failing{}).Register(mux)
	for _, req := range []*http.Request{
		httptest.NewRequest("GET", "/api/vehicles/VIN1/notifications", nil),
		httptest.NewRequest("PATCH", "/api/vehicles/VIN1/notifications/n1/read", nil),
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadGateway {
			t.Errorf("%s %s: want 502, got %d", req.Method, req.URL.Path, rec.Code)
		}
	}
}
