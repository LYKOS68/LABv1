package handlers

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimitMaxKeys deckelt die Map, damit viele wechselnde IPs den Speicher nicht füllen.
const rateLimitMaxKeys = 10000

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// RateLimiter erlaubt je Schlüssel (Bereich + IP) höchstens Attempts Versuche pro Window.
// Abgelaufene Einträge räumt jeder Aufruf selbst weg, deshalb braucht es keine Goroutine (TECH-STACK §7).
type RateLimiter struct {
	Attempts int
	Window   time.Duration

	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	now     func() time.Time // im Test austauschbar
}

func NewRateLimiter(attempts int, window time.Duration) *RateLimiter {
	return &RateLimiter{Attempts: attempts, Window: window, entries: map[string]*rateLimitEntry{}, now: time.Now}
}

// Allow zählt einen Versuch für key. Bei false nennt retryAfter die Wartezeit bis zum nächsten Versuch.
func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := rl.now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for k, e := range rl.entries {
		if now.Sub(e.windowStart) >= rl.Window {
			delete(rl.entries, k)
		}
	}

	e, ok := rl.entries[key]
	if !ok {
		if len(rl.entries) >= rateLimitMaxKeys {
			return false, rl.Window
		}
		rl.entries[key] = &rateLimitEntry{count: 1, windowStart: now}
		return true, 0
	}
	if e.count >= rl.Attempts {
		return false, rl.Window - now.Sub(e.windowStart)
	}
	e.count++
	return true, 0
}

// Limit setzt den Limiter vor einen Handler. scope trennt die Zähler, damit z. B. viele
// Login-Versuche nicht das Passwort-Zurücksetzen sperren.
func (rl *RateLimiter) Limit(scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := rl.Allow(scope + ":" + clientIP(r))
		if allowed {
			next(w, r)
			return
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		if isJSON(r) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too_many_requests"})
			return
		}
		http.Error(w, "Zu viele Versuche. Bitte warte ein paar Minuten.", http.StatusTooManyRequests)
	}
}

// clientIP nimmt die Adresse der direkten Verbindung. Hinter einem Proxy wäre das der Proxy;
// dann muss hier der weitergereichte Header ausgewertet werden.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
