package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LoginLimiter allows a few failed logins per key in a sliding window.
type LoginLimiter struct {
	Max    int
	Window time.Duration

	mu       sync.Mutex
	failures map[string][]time.Time
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{Max: 5, Window: 15 * time.Minute, failures: map[string][]time.Time{}}
}

func (l *LoginLimiter) prune(key string, now time.Time) []time.Time {
	kept := l.failures[key][:0]
	for _, t := range l.failures[key] {
		if now.Sub(t) < l.Window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = kept
	}
	return kept
}

// Allowed reports whether another attempt is allowed for every key.
func (l *LoginLimiter) Allowed(now time.Time, keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		if len(l.prune(k, now)) >= l.Max {
			return false
		}
	}
	return true
}

func (l *LoginLimiter) Fail(now time.Time, keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		l.failures[k] = append(l.prune(k, now), now)
	}
}

func (l *LoginLimiter) Reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.failures, k)
	}
}

// ClientIP returns the caller's IP. X-Forwarded-For is trusted only from a
// loopback peer (Caddy on the same host), so clients can't spoof it.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return host
}
