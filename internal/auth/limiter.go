package auth

import (
	"sync"
	"time"
)

// limiter locks an account after too many failed logins within a window.
// It is keyed by email, not IP: the API only sees the Next.js server's
// address, so per-IP limits would lock everyone out together.
type limiter struct {
	max    int
	window time.Duration

	mu      sync.Mutex
	entries map[string]*attempts
}

type attempts struct {
	count int
	start time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, entries: make(map[string]*attempts)}
}

// retryAfter reports how long the key stays locked (0 if it isn't).
func (l *limiter) retryAfter(key string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.entries[key]
	if !ok {
		return 0
	}
	end := a.start.Add(l.window)
	if !now.Before(end) {
		delete(l.entries, key)
		return 0
	}
	if a.count < l.max {
		return 0
	}
	return end.Sub(now)
}

func (l *limiter) fail(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) > 10_000 {
		l.prune(now)
	}
	a, ok := l.entries[key]
	if !ok || !now.Before(a.start.Add(l.window)) {
		l.entries[key] = &attempts{count: 1, start: now}
		return
	}
	a.count++
}

func (l *limiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// prune drops expired windows so the map can't grow without bound.
func (l *limiter) prune(now time.Time) {
	for k, a := range l.entries {
		if !now.Before(a.start.Add(l.window)) {
			delete(l.entries, k)
		}
	}
}
