package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	rl := NewRateLimiter(3, 10*time.Minute)
	rl.now = func() time.Time { return now }

	for i := range 3 {
		if ok, _ := rl.Allow("login:1.2.3.4"); !ok {
			t.Fatalf("attempt %d blocked, want allowed", i+1)
		}
	}
	ok, retry := rl.Allow("login:1.2.3.4")
	if ok {
		t.Fatal("attempt 4 allowed, want blocked")
	}
	if retry != 10*time.Minute {
		t.Fatalf("retryAfter = %v, want 10m", retry)
	}

	if ok, _ := rl.Allow("login:5.6.7.8"); !ok {
		t.Fatal("other IP blocked, want allowed")
	}
	if ok, _ := rl.Allow("forgot:1.2.3.4"); !ok {
		t.Fatal("other scope blocked, want allowed")
	}

	now = now.Add(10 * time.Minute)
	if ok, _ := rl.Allow("login:1.2.3.4"); !ok {
		t.Fatal("blocked after window, want allowed")
	}
}

func TestRateLimiterLimit(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	calls := 0
	h := rl.Limit("login", func(w http.ResponseWriter, r *http.Request) { calls++ })

	for range 2 {
		h(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/login", nil))
	}
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/login", nil))

	if calls != 1 {
		t.Fatalf("handler called %d times, want 1", calls)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing")
	}
}
