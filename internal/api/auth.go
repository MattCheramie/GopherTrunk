package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// AuthMode selects the auth policy applied to mutation endpoints.
//
//   - AuthModeAuto (default): the policy depends on the listener
//     binding. Loopback (127.0.0.1 / ::1) and any address in
//     AuthConfig.TrustedNetworks bypass the bearer-token check —
//     peer-cred via kernel-enforced reachability is treated as a
//     reasonable trust proxy on a single-host operator's box.
//     Anything else (0.0.0.0 / a public interface) requires a
//     valid Bearer token on every mutation request, and the
//     daemon refuses to start without a configured token.
//
//   - AuthModeRequired: every mutation request must carry a valid
//     Bearer token regardless of source, even loopback. Useful
//     when the daemon shares a host with untrusted users.
//
//   - AuthModeDisabled: bypass the bearer check entirely (the
//     legacy `allow_mutations: true` behaviour). Mutations are
//     wide open — for backwards-compatible single-host workflows
//     where the operator is the only one with shell access. The
//     daemon logs a warning at startup so this isn't accidentally
//     enabled in a hostile environment.
type AuthMode uint8

const (
	AuthModeAuto AuthMode = iota
	AuthModeRequired
	AuthModeDisabled
)

func (m AuthMode) String() string {
	switch m {
	case AuthModeAuto:
		return "auto"
	case AuthModeRequired:
		return "required"
	case AuthModeDisabled:
		return "disabled"
	default:
		return "?"
	}
}

// ParseAuthMode maps a config string into an AuthMode. Recognised
// values (case-insensitive):
//
//	""             → AuthModeDisabled (the new default — gophertrunk
//	                 is overwhelmingly deployed on closed LANs where
//	                 the bearer-token middleware is friction without
//	                 a corresponding threat model; opt back in by
//	                 setting "auto" or "required" explicitly)
//	"auto"         → AuthModeAuto
//	"required" / "on" / "true"    → AuthModeRequired
//	"disabled" / "off" / "false"  → AuthModeDisabled
//
// Unknown strings return AuthModeDisabled with ok=false so callers
// can warn without leaving the daemon in an ambiguous state.
func ParseAuthMode(s string) (AuthMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return AuthModeDisabled, true
	case "auto":
		return AuthModeAuto, true
	case "required", "on", "true":
		return AuthModeRequired, true
	case "disabled", "off", "false":
		return AuthModeDisabled, true
	default:
		return AuthModeDisabled, false
	}
}

// AuthConfig configures the bearer-token auth middleware.
type AuthConfig struct {
	// Mode picks the policy. See AuthMode for the trade-offs.
	Mode AuthMode
	// Token is the inline bearer token. Compared with
	// crypto/subtle.ConstantTimeCompare. Prefer TokenFile so the
	// token doesn't live in config.yaml — but inline is supported
	// for ephemeral / test setups.
	Token string
	// TokenFile is a path to a file containing the bearer token
	// (whitespace stripped). Read at startup; the daemon reloads it
	// on every request so operators can rotate tokens without a
	// restart. Empty disables file-based tokens.
	TokenFile string
	// TrustedNetworks is a list of CIDRs whose source addresses
	// bypass the bearer-token check under AuthModeAuto. Loopback
	// (127.0.0.1/32 and ::1/128) is implicitly trusted under
	// AuthModeAuto and does not need to be listed here.
	TrustedNetworks []string
}

// authState is the parsed, validated AuthConfig with the trusted-
// networks list pre-compiled into a slice of *net.IPNet, the token
// loaded into memory, and the listener address recorded so the auto
// policy can probe whether the daemon is bound to loopback.
type authState struct {
	mode     AuthMode
	token    atomic.Pointer[string] // nil = no token configured
	tokFile  string
	trusted  []*net.IPNet
	loopback bool // true when the listener binds to loopback only

	mu     sync.Mutex
	lastFS string // most recently observed token-file contents
}

// loopbackCIDRs are the prefixes the auto policy treats as
// implicitly trusted: IPv4 127/8 and IPv6 ::1.
var loopbackCIDRs = mustParseCIDRs([]string{
	"127.0.0.0/8",
	"::1/128",
})

func mustParseCIDRs(in []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(in))
	for _, s := range in {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			panic(fmt.Sprintf("api: invalid built-in CIDR %q: %v", s, err))
		}
		out = append(out, n)
	}
	return out
}

// newAuthState validates the AuthConfig and pre-computes everything
// the per-request middleware needs. listenAddr is the daemon's
// configured Addr — used to detect a loopback-only bind so the
// auto policy doesn't require a token on the safe default.
func newAuthState(cfg AuthConfig, listenAddr string) (*authState, error) {
	trusted := make([]*net.IPNet, 0, len(cfg.TrustedNetworks))
	for _, s := range cfg.TrustedNetworks {
		_, n, err := net.ParseCIDR(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("api: trusted_networks %q: %w", s, err)
		}
		trusted = append(trusted, n)
	}
	st := &authState{
		mode:     cfg.Mode,
		tokFile:  strings.TrimSpace(cfg.TokenFile),
		trusted:  trusted,
		loopback: bindsToLoopback(listenAddr),
	}
	if t := strings.TrimSpace(cfg.Token); t != "" {
		st.token.Store(&t)
	}
	if st.tokFile != "" {
		if err := st.reloadTokenFile(); err != nil {
			return nil, fmt.Errorf("api: token_file %q: %w", st.tokFile, err)
		}
	}
	// AuthModeRequired without any token configured is a config
	// error — there's no way to pass.
	if cfg.Mode == AuthModeRequired && st.token.Load() == nil {
		return nil, errors.New("api: auth.mode=required requires auth.token or auth.token_file")
	}
	// AuthModeAuto on a non-loopback bind also needs a token; the
	// loopback bypass doesn't apply when listening on 0.0.0.0.
	if cfg.Mode == AuthModeAuto && !st.loopback && st.token.Load() == nil && len(st.trusted) == 0 {
		return nil, errors.New("api: auth.mode=auto on a non-loopback listener requires auth.token, auth.token_file, or auth.trusted_networks")
	}
	return st, nil
}

// bindsToLoopback reports whether the listener address resolves to
// a loopback-only bind. ":8080" / "0.0.0.0:8080" / "[::]:8080" all
// reach external interfaces and return false; "127.0.0.1:8080" /
// "[::1]:8080" / "localhost:8080" return true.
func bindsToLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// No host part — fallback assumes a bare ":port" which
		// binds to all interfaces.
		return false
	}
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// reloadTokenFile reads tokFile, trims whitespace, and stores the
// result. Empty file = no token configured (caller validates).
func (s *authState) reloadTokenFile() error {
	if s.tokFile == "" {
		return nil
	}
	data, err := os.ReadFile(s.tokFile)
	if err != nil {
		return err
	}
	tok := strings.TrimSpace(string(data))
	s.mu.Lock()
	s.lastFS = tok
	s.mu.Unlock()
	if tok == "" {
		s.token.Store(nil)
		return nil
	}
	s.token.Store(&tok)
	return nil
}

// authorize returns nil when the request is allowed through to the
// handler. Returns a 401/403 status code + reason otherwise.
func (s *authState) authorize(r *http.Request) (int, string) {
	switch s.mode {
	case AuthModeDisabled:
		return 0, ""
	case AuthModeRequired:
		return s.checkToken(r)
	case AuthModeAuto:
		if s.sourceTrusted(r) {
			return 0, ""
		}
		return s.checkToken(r)
	default:
		return http.StatusInternalServerError, "auth: invalid mode"
	}
}

func (s *authState) sourceTrusted(r *http.Request) bool {
	// Implicit loopback trust under auto: if the listener binds to
	// loopback only, every request is loopback-sourced by
	// definition, and there's no kernel path for off-host
	// requests to reach this socket.
	if s.loopback {
		return true
	}
	ip := remoteIP(r)
	if ip == nil {
		return false
	}
	for _, n := range loopbackCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	for _, n := range s.trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *authState) checkToken(r *http.Request) (int, string) {
	// Reload the token from disk on every request — keeps rotation
	// cheap (a single read syscall) and avoids the need for a
	// SIGHUP handler in the daemon.
	if s.tokFile != "" {
		if err := s.reloadTokenFile(); err != nil {
			return http.StatusInternalServerError, "auth: token_file unreadable"
		}
	}
	want := s.token.Load()
	if want == nil || *want == "" {
		return http.StatusForbidden, "auth: no token configured (set api.auth.token or api.auth.token_file)"
	}
	got, ok := bearerToken(r)
	if !ok {
		return http.StatusUnauthorized, "auth: missing Authorization: Bearer header"
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(*want)) != 1 {
		return http.StatusUnauthorized, "auth: invalid token"
	}
	return 0, ""
}

// bearerToken extracts the token from the Authorization header. The
// header must read "Bearer <token>" (RFC 6750); leading whitespace
// is tolerated.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) && !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return "", false
	}
	// Trim the prefix case-insensitively.
	if len(h) <= len(prefix) {
		return "", false
	}
	tok := strings.TrimSpace(h[len(prefix):])
	if tok == "" {
		return "", false
	}
	return tok, true
}

// remoteIP returns the source IP of the request. Looks at
// RemoteAddr only — explicitly does not honour X-Forwarded-For,
// since the loopback bypass shouldn't be forgeable by a hostile
// upstream proxy.
func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// In tests RemoteAddr is sometimes a bare host.
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

// canMutate reports whether the supplied request would be allowed
// through the auth middleware. Used by GET /api/v1/mutations to
// tell clients up-front whether mutation routes will accept their
// credentials.
func (s *authState) canMutate(r *http.Request) bool {
	status, _ := s.authorize(r)
	return status == 0
}
