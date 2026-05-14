package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS_Disabled(t *testing.T) {
	// When AllowedOrigins is empty the middleware is a pass-through.
	cfg := CORSConfig{}
	if cfg.enabled() {
		t.Fatalf("enabled() = true; want false")
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(cfg, inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q; want empty", got)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"null", "http://laptop.local:8000"}}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := corsMiddleware(cfg, inner)

	cases := []struct {
		origin string
		want   string
	}{
		{"null", "null"},
		{"http://laptop.local:8000", "http://laptop.local:8000"},
		{"HTTP://Laptop.local:8000", "HTTP://Laptop.local:8000"}, // case-insensitive match, echo as sent
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Origin", tc.origin)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("origin=%q status=%d", tc.origin, rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tc.want {
			t.Errorf("origin=%q Allow-Origin=%q want %q", tc.origin, got, tc.want)
		}
		if got := rr.Header().Get("Vary"); got != "Origin" {
			t.Errorf("origin=%q Vary=%q want Origin", tc.origin, got)
		}
	}
}

func TestCORS_BlockedOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"null"}}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(cfg, inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// The inner handler still runs (CORS is browser-side enforcement),
	// but the response carries no Allow-Origin so the browser will
	// reject the response.
	if rr.Code != http.StatusOK {
		t.Errorf("status=%d want 200", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q; want empty for blocked origin", got)
	}
}

func TestCORS_Wildcard(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := corsMiddleware(cfg, inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://anything.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://anything.example" {
		t.Errorf("Allow-Origin = %q; want echoed origin", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"null"}}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("inner handler called on preflight")
		w.WriteHeader(http.StatusTeapot)
	})
	h := corsMiddleware(cfg, inner)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/talkgroups/123", nil)
	req.Header.Set("Origin", "null")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight status=%d want 204", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "null" {
		t.Errorf("Allow-Origin=%q want null", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "PATCH") {
		t.Errorf("Allow-Methods=%q missing PATCH", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Errorf("Allow-Headers=%q missing Authorization", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	// Same-origin or non-browser requests have no Origin header.
	// The middleware should still pass them through without adding
	// any CORS headers, leaving the inner handler to do its job.
	cfg := CORSConfig{AllowedOrigins: []string{"null"}}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := corsMiddleware(cfg, inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status=%d want 200", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin=%q want empty when Origin header absent", got)
	}
}
