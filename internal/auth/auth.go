// Package auth handles sign-in: users with hashed passwords, and opaque
// session tokens kept in memory.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
}

var ErrInvalidCredentials = errors.New("invalid email or password")

// ErrTooManyAttempts means the account is temporarily locked after repeated
// failed logins. Use errors.As to get the RetryAfter duration.
var ErrTooManyAttempts = errors.New("too many failed sign-in attempts")

type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string {
	return fmt.Sprintf("%v; try again in %v", ErrTooManyAttempts, e.RetryAfter.Round(time.Second))
}
func (e *LockedError) Unwrap() error { return ErrTooManyAttempts }

const (
	maxFailedLogins = 5
	lockoutWindow   = 15 * time.Minute
)

type session struct {
	userID  string
	expires time.Time
}

type Service struct {
	ttl   time.Duration
	now   func() time.Time
	users map[string]User // by lower-cased email
	byID  map[string]User
	// dummyHash is checked for unknown emails so failed logins take the same
	// time whether or not the account exists.
	dummyHash string
	limits    *limiter

	mu       sync.Mutex
	sessions map[string]session // by SHA-256 of the token, never the token itself
}

func NewService(users []User, ttl time.Duration) *Service {
	s := &Service{
		ttl:      ttl,
		now:      time.Now,
		users:    make(map[string]User, len(users)),
		byID:     make(map[string]User, len(users)),
		sessions: make(map[string]session),
		limits:   newLimiter(maxFailedLogins, lockoutWindow),
	}
	for _, u := range users {
		s.users[strings.ToLower(u.Email)] = u
		s.byID[u.ID] = u
		if s.dummyHash == "" {
			s.dummyHash = u.PasswordHash
		}
	}
	return s
}

// Login checks credentials and starts a session, returning its token.
// Repeated failures for the same email lock it for a while (*LockedError),
// whether or not the account exists.
func (s *Service) Login(email, password string) (string, User, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	now := s.now()
	if wait := s.limits.retryAfter(key, now); wait > 0 {
		return "", User{}, &LockedError{RetryAfter: wait}
	}
	u, ok := s.users[key]
	if !ok {
		VerifyPassword(s.dummyHash, password)
		s.limits.fail(key, now)
		return "", User{}, ErrInvalidCredentials
	}
	if !VerifyPassword(u.PasswordHash, password) {
		s.limits.fail(key, now)
		return "", User{}, ErrInvalidCredentials
	}
	s.limits.reset(key)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", User{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[digest(token)] = session{userID: u.ID, expires: s.now().Add(s.ttl)}
	return token, u, nil
}

// Authenticate returns the user a live session token belongs to.
func (s *Service) Authenticate(token string) (User, bool) {
	if token == "" {
		return User{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := digest(token)
	sess, ok := s.sessions[key]
	if !ok {
		return User{}, false
	}
	if !s.now().Before(sess.expires) {
		delete(s.sessions, key)
		return User{}, false
	}
	u, ok := s.byID[sess.userID]
	return u, ok
}

// Logout ends a session. Unknown tokens are ignored.
func (s *Service) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, digest(token))
}

func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
