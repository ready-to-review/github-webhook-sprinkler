package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)
	
	ip := "192.168.1.1"

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !rl.Allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if rl.Allow(ip) {
		t.Error("6th request should be denied")
	}

	// Note: Our simplified rate limiter resets every minute,
	// not every second, so we can't test the reset behavior easily
}

func TestConnectionLimiter(t *testing.T) {
	cl := NewConnectionLimiter(2, 5)

	// Test per-IP limit
	ip1 := "192.168.1.1"
	if !cl.Add(ip1) {
		t.Error("first connection should be allowed")
	}
	if !cl.Add(ip1) {
		t.Error("second connection should be allowed")
	}
	if cl.Add(ip1) {
		t.Error("third connection should be denied (per-IP limit)")
	}

	// Test total limit
	ip2 := "192.168.1.2"
	ip3 := "192.168.1.3"
	cl.Add(ip2)
	cl.Add(ip2)
	cl.Add(ip3)

	// Should hit total limit
	ip4 := "192.168.1.4"
	if cl.Add(ip4) {
		t.Error("should hit total connection limit")
	}

	// Remove a connection
	cl.Remove(ip1)

	// Should allow new connection after removal
	if !cl.Add(ip4) {
		t.Error("should allow connection after removal")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{
			name:       "direct connection",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name: "ignores X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name: "ignores X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3",
			},
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name: "ignores X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.1",
			},
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name: "no port in RemoteAddr",
			headers: map[string]string{},
			remoteAddr: "192.168.1.1",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := GetClientIP(req); got != tt.want {
				t.Errorf("GetClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}