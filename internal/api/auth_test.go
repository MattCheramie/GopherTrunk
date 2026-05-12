package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestParseAuthMode(t *testing.T) {
	cases := []struct {
		in   string
		want AuthMode
		ok   bool
	}{
		{"", AuthModeAuto, true},
		{"auto", AuthModeAuto, true},
		{"AUTO", AuthModeAuto, true},
		{" auto ", AuthModeAuto, true},
		{"required", AuthModeRequired, true},
		{"on", AuthModeRequired, true},
		{"true", AuthModeRequired, true},
		{"disabled", AuthModeDisabled, true},
		{"off", AuthModeDisabled, true},
		{"false", AuthModeDisabled, true},
		{"nonsense", AuthModeAuto, false},
	}
	for _, c := range cases {
		got, ok := ParseAuthMode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseAuthMode(%q) = (%v, %v), want (%v, %v)",
				c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestBindsToLoopback(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"127.0.0.1:0", true},
		{"[::1]:8080", true},
		{"localhost:8080", true},
		{"0.0.0.0:8080", false},
		{"[::]:8080", false},
		{":8080", false},
		{"192.168.1.10:8080", false},
		{"example.com:8080", false},
	}
	for _, c := range cases {
		got := bindsToLoopback(c.addr)
		if got != c.want {
			t.Errorf("bindsToLoopback(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}

func TestNewAuthState_RejectsRequiredWithoutToken(t *testing.T) {
	_, err := newAuthState(AuthConfig{Mode: AuthModeRequired}, "127.0.0.1:8080")
	if err == nil {
		t.Fatal("expected error for required mode without token")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("error = %q, want mention of 'required'", err)
	}
}

func TestNewAuthState_RejectsAutoOnPublicWithoutToken(t *testing.T) {
	_, err := newAuthState(AuthConfig{Mode: AuthModeAuto}, "0.0.0.0:8080")
	if err == nil {
		t.Fatal("expected error for auto mode on non-loopback without token")
	}
}

func TestNewAuthState_AcceptsAutoOnLoopbackWithoutToken(t *testing.T) {
	st, err := newAuthState(AuthConfig{Mode: AuthModeAuto}, "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st == nil || st.mode != AuthModeAuto {
		t.Errorf("expected AuthModeAuto state, got %+v", st)
	}
}

func TestNewAuthState_AcceptsDisabledOnAnyBind(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:8080", "127.0.0.1:8080", ":8080"} {
		_, err := newAuthState(AuthConfig{Mode: AuthModeDisabled}, addr)
		if err != nil {
			t.Errorf("addr=%q: unexpected error: %v", addr, err)
		}
	}
}

func TestAuthorize_DisabledModeAllowsAll(t *testing.T) {
	st, err := newAuthState(AuthConfig{Mode: AuthModeDisabled}, "0.0.0.0:8080")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("expected allowed, got status=%d", status)
	}
}

func TestAuthorize_AutoLoopbackBypass(t *testing.T) {
	// Loopback listener — any request bypasses auth even without a token.
	st, err := newAuthState(AuthConfig{Mode: AuthModeAuto}, "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.RemoteAddr = "192.168.1.50:12345" // irrelevant; loopback bind trusts everyone
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("expected loopback-listener bypass, got status=%d", status)
	}
}

func TestAuthorize_AutoPublicRequiresToken(t *testing.T) {
	st, err := newAuthState(AuthConfig{Mode: AuthModeAuto, Token: "secret"}, "0.0.0.0:8080")
	if err != nil {
		t.Fatal(err)
	}

	// No header → 401
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	if status, _ := st.authorize(req); status != http.StatusUnauthorized {
		t.Errorf("no header: status=%d, want 401", status)
	}

	// Wrong token → 401
	req.Header.Set("Authorization", "Bearer not-the-token")
	if status, _ := st.authorize(req); status != http.StatusUnauthorized {
		t.Errorf("wrong token: status=%d, want 401", status)
	}

	// Correct token → 0 (allowed)
	req.Header.Set("Authorization", "Bearer secret")
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("correct token: status=%d, want 0", status)
	}
}

func TestAuthorize_TrustedNetworkBypass(t *testing.T) {
	st, err := newAuthState(AuthConfig{
		Mode:            AuthModeAuto,
		Token:           "secret",
		TrustedNetworks: []string{"192.168.0.0/16"},
	}, "0.0.0.0:8080")
	if err != nil {
		t.Fatal(err)
	}

	// Source inside the trusted network — no token needed.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("trusted network: status=%d, want 0", status)
	}

	// Source outside the trusted network — token required.
	req.RemoteAddr = "10.0.0.5:12345"
	if status, _ := st.authorize(req); status != http.StatusUnauthorized {
		t.Errorf("untrusted network without token: status=%d, want 401", status)
	}
}

func TestAuthorize_RequiredModeIgnoresLoopback(t *testing.T) {
	// Required mode always needs a token, even from loopback.
	st, err := newAuthState(AuthConfig{Mode: AuthModeRequired, Token: "secret"}, "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	if status, _ := st.authorize(req); status != http.StatusUnauthorized {
		t.Errorf("loopback without token in required mode: status=%d, want 401", status)
	}
	req.Header.Set("Authorization", "Bearer secret")
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("loopback with token in required mode: status=%d, want 0", status)
	}
}

func TestAuthorize_TokenFileRotation(t *testing.T) {
	dir := t.TempDir()
	tokFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokFile, []byte("first-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := newAuthState(AuthConfig{
		Mode:      AuthModeRequired,
		TokenFile: tokFile,
	}, "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}

	// Old token accepted.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/abc/end", nil)
	req.Header.Set("Authorization", "Bearer first-token")
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("first token: status=%d, want 0", status)
	}

	// Rotate the file.
	if err := os.WriteFile(tokFile, []byte("second-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Old token rejected on the next request — no SIGHUP needed.
	if status, _ := st.authorize(req); status != http.StatusUnauthorized {
		t.Errorf("old token after rotation: status=%d, want 401", status)
	}

	// New token accepted.
	req.Header.Set("Authorization", "Bearer second-token")
	if status, _ := st.authorize(req); status != 0 {
		t.Errorf("second token: status=%d, want 0", status)
	}
}

func TestBearerToken_Parsing(t *testing.T) {
	cases := []struct {
		header string
		want   string
		ok     bool
	}{
		{"", "", false},
		{"Bearer abc", "abc", true},
		{"bearer abc", "abc", true},   // case-insensitive prefix
		{"BEARER xyz", "xyz", true},   // RFC 6750 §2.1 is case-insensitive
		{"Bearer ", "", false},
		{"Bearer    secret   ", "secret", true},
		{"Basic abc", "", false},
		{"Token abc", "", false},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.header != "" {
			req.Header.Set("Authorization", c.header)
		}
		got, ok := bearerToken(req)
		if got != c.want || ok != c.ok {
			t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)",
				c.header, got, ok, c.want, c.ok)
		}
	}
}

func TestLegacyAllowMutations_MapsToDisabled(t *testing.T) {
	// NewServer with AllowMutations: true and no Auth config should
	// log a deprecation warning and treat the daemon as auth disabled.
	bus := events.NewBus(8)
	defer bus.Close()
	s, err := NewServer(ServerOptions{
		Addr:           "127.0.0.1:0",
		Bus:            bus,
		AllowMutations: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.auth.mode != AuthModeDisabled {
		t.Errorf("auth mode = %v, want AuthModeDisabled (legacy migration)", s.auth.mode)
	}
}
