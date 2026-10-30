package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxyClientIP(t *testing.T) {
	p, err := NewProxyPolicy("10.0.0.1/32,10.0.0.2/32,2001:db8:1::/64")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		remote string
		xff string
		want string
	}{
		{"direct", "198.51.100.1:1234", "", "198.51.100.1"},
		{"direct spoof", "198.51.100.1:1234", "203.0.113.1", "198.51.100.1"},
		{"termination", "10.0.0.1:1234", "198.51.100.1", "198.51.100.1"},
		{"trusted chain", "10.0.0.1:1234", "198.51.100.1, 10.0.0.2", "198.51.100.1"},
		{"forged leftmost", "10.0.0.1:1234", "203.0.113.1, 198.51.100.1", "198.51.100.1"},
		{"missing", "10.0.0.1:1234", "", "10.0.0.1"},
		{"invalid", "10.0.0.1:1234", "garbage, 198.51.100.1", "10.0.0.1"},
		{"empty hop", "10.0.0.1:1234", "198.51.100.1,", "10.0.0.1"},
		{"port", "10.0.0.1:1234", "198.51.100.1:80", "10.0.0.1"},
		{"ipv6", "[2001:db8:1::1]:1234", "2001:db8:2::1", "2001:db8:2::1"},
		{"mapped", "[::ffff:198.51.100.1]:1234", "203.0.113.1", "198.51.100.1"},
		{"zone", "10.0.0.1:1234", "fe80::1%eth0", "10.0.0.1"},
		{"oversized", "10.0.0.1:1234", strings.Repeat("1", 4097), "10.0.0.1"},
		{"too many hops", "10.0.0.1:1234", strings.Repeat("10.0.0.2,", 32) + "198.51.100.1", "10.0.0.1"},
		{"invalid peer", "invalid", "198.51.100.1", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/login", nil)
			r.RemoteAddr = tt.remote
			r.Header.Set("X-Forwarded-For", tt.xff)
			r.Header.Set("Forwarded", "for=203.0.113.99;proto=https")
			r.Header.Set("X-Real-IP", "203.0.113.99")
			if got := p.clientIP(r); got != tt.want {
				t.Fatalf("client IP = %q, want %q", got, tt.want)
			}
		})
	}
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.RemoteAddr = "10.0.0.1:80"
	r.Header.Add("X-Forwarded-For", "203.0.113.1")
	r.Header.Add("X-Forwarded-For", "198.51.100.1")
	if got := p.clientIP(r); got != "198.51.100.1" {
		t.Fatalf("multiple headers: %s", got)
	}
	var noTrust *ProxyPolicy
	if got := noTrust.clientIP(r); got != "10.0.0.1" {
		t.Fatalf("default trusted a header: %s", got)
	}
}

func TestProxyConfiguration(t *testing.T) {
	for _, value := range []string{"invalid", "10.0.0.1", "10.0.0.1/32,", "0.0.0.0/0", "::/0", "::ffff:10.0.0.1/128"} {
		if _, err := NewProxyPolicy(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if _, err := NewProxyPolicy(""); err != nil {
		t.Fatal(err)
	}
}

func TestProxyRateLimitIsolation(t *testing.T) {
	p, err := NewProxyPolicy("10.0.0.1/32")
	if err != nil {
		t.Fatal(err)
	}
	rl := NewRateLimiter(1, time.Minute)
	rl.ProxyPolicy = p
	h := rl.Limit("login", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, tt := range []struct {
		remote string
		xff string
		status int
	}{
		{"10.0.0.1:80", "198.51.100.1", 204},
		{"10.0.0.1:80", "198.51.100.2", 204},
		{"10.0.0.1:80", "203.0.113.1, 198.51.100.1", 429},
		{"198.51.100.3:80", "203.0.113.1", 204},
		{"198.51.100.3:81", "203.0.113.2", 429},
	} {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.RemoteAddr = tt.remote
		r.Header.Set("X-Forwarded-For", tt.xff)
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != tt.status {
			t.Fatalf("%s / %s: status %d, want %d", tt.remote, tt.xff, w.Code, tt.status)
		}
	}
}

func TestSessionCookieSecurity(t *testing.T) {
	for _, tt := range []struct {
		name string
		local bool
		tls bool
		proto string
		secure bool
	}{
		{"online termination", false, false, "https", true},
		{"online forged downgrade", false, false, "http", true},
		{"online direct TLS", false, true, "http", true},
		{"local HTTP", true, false, "https", false},
		{"local TLS", true, true, "http", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handlers{AllowInsecureLocalHTTP: tt.local}
			r := httptest.NewRequest(http.MethodPost, "/logout", nil)
			r.Header.Set("X-Forwarded-Proto", tt.proto)
			r.Header.Set("Forwarded", "proto=" + tt.proto)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			issued := httptest.NewRecorder()
			h.setSessionCookie(issued, r, "test-token", int(sessionTTL.Seconds()))
			issuedCookies := issued.Result().Cookies()
			if len(issuedCookies) != 1 {
				t.Fatalf("issued cookies: %v", issuedCookies)
			}
			created := issuedCookies[0]
			if created.Name != sessionCookie || created.Value != "test-token" || created.Secure != tt.secure || !created.HttpOnly || created.SameSite != http.SameSiteLaxMode || created.Path != "/" || created.MaxAge != int(sessionTTL.Seconds()) {
				t.Fatalf("unexpected issued cookie: %v", created)
			}
			w := httptest.NewRecorder()
			h.Logout(w, r)
			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies: %v", cookies)
			}
			c := cookies[0]
			if c.Secure != tt.secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != -1 {
				t.Fatalf("unexpected cookie: %v", c)
			}
		})
	}
}
