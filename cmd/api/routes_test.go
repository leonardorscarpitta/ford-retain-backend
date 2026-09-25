package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAccessControl exercises the real router: who can reach what.
func TestAccessControl(t *testing.T) {
	users, err := seedUsers(1000)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(time.Now(), users)

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
		rec := do("POST", "/api/auth/login", "", `{"email":"`+email+`","password":"`+demoPassword+`"}`)
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
		{"own telemetry", "GET", "/api/vehicles/" + f150VIN + "/telemetry", ana, 200},
		{"other owner's telemetry", "GET", "/api/vehicles/" + rangerVIN + "/telemetry", ana, 404},
		{"other owner's vehicle", "GET", "/api/vehicles/" + f150VIN, bruno, 404},
		{"other owner's appointments", "GET", "/api/vehicles/" + f150VIN + "/appointments", bruno, 404},
		{"cancel other owner's appointment", "DELETE", "/api/vehicles/" + f150VIN + "/appointments/apt-seed-1", bruno, 404},
		{"mark other owner's alert read", "PATCH", "/api/vehicles/" + f150VIN + "/notifications/n1/read", bruno, 404},
		{"own alert", "PATCH", "/api/vehicles/" + f150VIN + "/notifications/n1/read", ana, 200},
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
	if len(garage) != 1 || garage[0].VIN != rangerVIN {
		t.Fatalf("bruno's garage: want only the Ranger, got %v", garage)
	}

	do("POST", "/api/auth/logout", ana, "")
	if rec := do("GET", "/api/vehicles", ana, ""); rec.Code != 401 {
		t.Fatalf("after logout: want 401, got %d", rec.Code)
	}
}
