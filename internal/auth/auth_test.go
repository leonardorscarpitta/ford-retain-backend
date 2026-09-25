package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ford-retain/backend/internal/platform/identity"
)

func testService(t *testing.T) *Service {
	t.Helper()
	hash, err := HashPassword("secret", 1000) // low cost keeps tests fast
	if err != nil {
		t.Fatal(err)
	}
	return NewService([]User{{ID: "u1", Email: "Ana@Example.com", Name: "Ana", PasswordHash: hash}}, time.Hour)
}

func TestPassword(t *testing.T) {
	hash, err := HashPassword("correct horse", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "correct horse") {
		t.Error("right password rejected")
	}
	if VerifyPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
	if VerifyPassword("garbage", "correct horse") {
		t.Error("malformed hash accepted")
	}
	other, _ := HashPassword("correct horse", 1000)
	if other == hash {
		t.Error("same password hashed twice should differ (salt)")
	}
}

func TestSessions(t *testing.T) {
	s := testService(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	if _, _, err := s.Login("ana@example.com", "nope"); err != ErrInvalidCredentials {
		t.Fatalf("wrong password: want ErrInvalidCredentials, got %v", err)
	}
	if _, _, err := s.Login("nobody@example.com", "secret"); err != ErrInvalidCredentials {
		t.Fatalf("unknown email: want ErrInvalidCredentials, got %v", err)
	}
	token, u, err := s.Login("  ANA@example.com ", "secret")
	if err != nil || u.ID != "u1" {
		t.Fatalf("login: %v %v", u, err)
	}
	if got, ok := s.Authenticate(token); !ok || got.ID != "u1" {
		t.Fatal("fresh session should authenticate")
	}
	if _, ok := s.Authenticate(token + "x"); ok {
		t.Fatal("tampered token authenticated")
	}

	now = now.Add(2 * time.Hour)
	if _, ok := s.Authenticate(token); ok {
		t.Fatal("expired session authenticated")
	}

	now = now.Add(-2 * time.Hour)
	token, _, _ = s.Login("ana@example.com", "secret")
	s.Logout(token)
	if _, ok := s.Authenticate(token); ok {
		t.Fatal("session survived logout")
	}
}

func TestHTTP(t *testing.T) {
	h := NewHandler(testService(t))
	public := http.NewServeMux()
	h.RegisterPublic(public)
	private := http.NewServeMux()
	h.Register(private)
	private.HandleFunc("GET /api/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(identity.UserID(r.Context())))
	})
	public.Handle("/api/", h.RequireUser(private))

	do := func(method, path, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		public.ServeHTTP(rec, req)
		return rec
	}

	if rec := do("POST", "/api/auth/login", `{"email":"ana@example.com","password":"bad"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: want 401, got %d", rec.Code)
	}
	rec := do("POST", "/api/auth/login", `{"email":"ana@example.com","password":"secret"}`, "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "pbkdf2") {
		t.Fatalf("login: status %d, body must not leak the hash: %s", rec.Code, rec.Body)
	}
	token := strings.Split(strings.Split(rec.Body.String(), `"token":"`)[1], `"`)[0]

	if rec := do("GET", "/api/whoami", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}
	if rec := do("GET", "/api/whoami", "", token); rec.Body.String() != "u1" {
		t.Fatalf("identity not in context: %q", rec.Body)
	}
	if rec := do("GET", "/api/auth/me", "", token); rec.Code != http.StatusOK {
		t.Fatalf("me: want 200, got %d", rec.Code)
	}
	do("POST", "/api/auth/logout", "", token)
	if rec := do("GET", "/api/auth/me", "", token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: want 401, got %d", rec.Code)
	}
}

func TestLockout(t *testing.T) {
	s := testService(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	for i := 0; i < maxFailedLogins; i++ {
		if _, _, err := s.Login("ana@example.com", "wrong"); err != ErrInvalidCredentials {
			t.Fatalf("attempt %d: want ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	// Locked now — even the right password is refused, without checking it.
	_, _, err := s.Login("ANA@example.com", "secret")
	var locked *LockedError
	if !errors.As(err, &locked) || !errors.Is(err, ErrTooManyAttempts) || locked.RetryAfter != lockoutWindow {
		t.Fatalf("want lockout for the full window, got %v", err)
	}

	// Unknown emails lock the same way, so lockouts don't reveal accounts.
	for i := 0; i < maxFailedLogins; i++ {
		s.Login("ghost@example.com", "x")
	}
	if _, _, err := s.Login("ghost@example.com", "x"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("unknown email: want lockout, got %v", err)
	}

	// The window passes: the account works again and a success clears the count.
	now = now.Add(lockoutWindow)
	if _, _, err := s.Login("ana@example.com", "secret"); err != nil {
		t.Fatalf("after window: %v", err)
	}
	for i := 0; i < maxFailedLogins-1; i++ {
		s.Login("ana@example.com", "wrong")
	}
	if _, _, err := s.Login("ana@example.com", "secret"); err != nil {
		t.Fatalf("below the limit a correct password must work: %v", err)
	}
}

func TestLockoutHTTP(t *testing.T) {
	h := NewHandler(testService(t))
	mux := http.NewServeMux()
	h.RegisterPublic(mux)
	var rec *httptest.ResponseRecorder
	for i := 0; i <= maxFailedLogins; i++ {
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"ana@example.com","password":"bad"}`)))
	}
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("want 429 with Retry-After 900, got %d %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}
