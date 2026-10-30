package main

import "testing"

func TestHTTPListenAddr(t *testing.T) {
	for _, tt := range []struct {
		url string
		local bool
		want string
	}{
		{"https://lab.example", false, ":8080"},
		{"http://lab.example", false, ""},
		{"http://localhost:8080", false, ""},
		{"http://localhost:8080", true, "127.0.0.1:8080"},
		{"http://127.0.0.1:8080", true, "127.0.0.1:8080"},
		{"http://[::1]:8080", true, "127.0.0.1:8080"},
		{"http://lab.example", true, ""},
		{"http://localhost.evil.example", true, ""},
		{"https://localhost", true, ""},
		{"https://user:password@lab.example", false, ""},
		{"/relative", false, ""},
		{"http://[", true, ""},
	} {
		t.Run(tt.url, func(t *testing.T) {
			got, err := httpListenAddr(tt.url, "8080", tt.local)
			if (err != nil) != (tt.want == "") || got != tt.want {
				t.Fatalf("address = %q, err = %v, want %q", got, err, tt.want)
			}
		})
	}
}
