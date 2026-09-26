package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/fordapi"
	"github.com/ford-retain/backend/internal/seed"
)

// TestAccessControl exercises the real router, who can reach what, once
// with seed data and once reading through the Ford client (against the mock
// Ford API over HTTP).
func TestAccessControl(t *testing.T) {
	now := time.Now()
	t.Run("seed data", func(t *testing.T) { testAccessControl(t, seedSources(now)) })
	t.Run("Ford platform", func(t *testing.T) {
		mock := httptest.NewServer(fordapi.NewMock("k", fordapi.MockData{
			Vehicles:  seed.Vehicles(),
			Telemetry: seed.Telemetry(now),
			Alerts:    seed.Notifications(now),
			Dealers:   seed.Dealerships(),
		}))
		defer mock.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		src := fordSources(ctx, fordapi.New(mock.URL, "k"), time.Hour)
		if len(src.dealers.All()) != 24 {
			t.Fatalf("dealer directory not loaded at startup")
		}
		testAccessControl(t, src)
	})
}

func testAccessControl(t *testing.T, src sources) {
	users, err := seed.Users(1000)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(users, src, appointments.NewMemoryRepo(seed.Appointments(time.Now())))

	do := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	login := func(email string) string {
		rec := do("POST", "/api/auth/login", "", `{"email":"`+email+`","password":"`+seed.DemoPassword+`"}`)
		var out struct{ Token string }
		if rec.Code != http.StatusOK || json.NewDecoder(rec.Body).Decode(&out) != nil || out.Token == "" {
			t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
		}
		return out.Token
	}
	ana, bruno := login("ana@example.com"), login("bruno@example.com")

	tests := []struct {
		name, method, path, token string
		want                      int
	}{
		{"health is public", "GET", "/healthz", "", 200},
		{"API needs a session", "GET", "/api/vehicles", "", 401},
		{"dealerships need a session", "GET", "/api/dealerships", "", 401},
		{"bad token", "GET", "/api/vehicles", "nope", 401},
		{"own garage", "GET", "/api/vehicles", ana, 200},
		{"own telemetry", "GET", "/api/vehicles/" + seed.F150VIN + "/telemetry", ana, 200},
		{"other owner's telemetry", "GET", "/api/vehicles/" + seed.RangerVIN + "/telemetry", ana, 404},
		{"other owner's vehicle", "GET", "/api/vehicles/" + seed.F150VIN, bruno, 404},
		{"other owner's appointments", "GET", "/api/vehicles/" + seed.F150VIN + "/appointments", bruno, 404},
		{"cancel other owner's appointment", "DELETE", "/api/vehicles/" + seed.F150VIN + "/appointments/apt-seed-1", bruno, 404},
		{"mark other owner's alert read", "PATCH", "/api/vehicles/" + seed.F150VIN + "/notifications/n1/read", bruno, 404},
		{"own alert", "PATCH", "/api/vehicles/" + seed.F150VIN + "/notifications/n1/read", ana, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(tt.method, tt.path, tt.token, ""); rec.Code != tt.want {
				t.Fatalf("want %d, got %d: %s", tt.want, rec.Code, rec.Body)
			}
		})
	}

	var garage []struct{ VIN string }
	json.NewDecoder(do("GET", "/api/vehicles", bruno, "").Body).Decode(&garage)
	if len(garage) != 1 || garage[0].VIN != seed.RangerVIN {
		t.Fatalf("bruno's garage: want only the Ranger, got %v", garage)
	}

	do("POST", "/api/auth/logout", ana, "")
	if rec := do("GET", "/api/vehicles", ana, ""); rec.Code != 401 {
		t.Fatalf("after logout: want 401, got %d", rec.Code)
	}
}
