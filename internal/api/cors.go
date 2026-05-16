package api

import (
	"net/http"
	"strings"
)

// CORSConfig configures the cross-origin middleware. AllowedOrigins
// is the exact list of values the daemon will echo back in
// Access-Control-Allow-Origin. The special value "*" matches any
// origin. The literal "null" matches the Origin header browsers
// send for file:// loads.
//
// When AllowedOrigins is empty the daemon's permissive default
// (CORS open to any origin) applies — closed-LAN setups don't have
// to opt into CORS to load the web SPA from file:// or a sibling
// static server. Operators who run on a hostile network override
// this list to clamp it back down.
type CORSConfig struct {
	AllowedOrigins []string
}

// effectiveOrigins returns the configured allow-list, substituting
// the permissive default (["*"]) when none was supplied.
func (c CORSConfig) effectiveOrigins() []string {
	if len(c.AllowedOrigins) == 0 {
		return []string{"*"}
	}
	return c.AllowedOrigins
}

// originAllowed reports whether the supplied request Origin is on
// the allow-list and returns the exact value to echo back. The
// returned string is empty when the origin is not allowed.
func (c CORSConfig) originAllowed(origin string) (string, bool) {
	if origin == "" {
		return "", false
	}
	for _, allowed := range c.effectiveOrigins() {
		if allowed == "*" {
			// Wildcard: echo the actual origin back so credentialed
			// requests still work. Browsers reject "*" combined with
			// Access-Control-Allow-Credentials.
			return origin, true
		}
		if strings.EqualFold(allowed, origin) {
			return origin, true
		}
	}
	return "", false
}

// enabled is true whenever the middleware should run. With the
// permissive default in effect this is always true; an operator who
// wants no CORS headers at all must explicitly set AllowedOrigins
// to a non-matching value.
func (c CORSConfig) enabled() bool { return true }

// IsDefaultPermissive reports whether the runtime is operating on
// the empty-config-means-allow-all default. Surfaced so the daemon
// can warn at startup when a non-loopback bind is combined with the
// permissive default.
func (c CORSConfig) IsDefaultPermissive() bool {
	return len(c.AllowedOrigins) == 0
}

// corsMiddleware wraps the inner handler with CORS headers when the
// request's Origin matches one of the configured allow-list entries.
// Preflight (OPTIONS) requests short-circuit to 204 with the
// appropriate Allow-* headers; everything else is delegated to next
// after the headers have been added.
//
// The middleware is intentionally permissive about headers and
// methods because the API is small and well-bounded: every mutation
// uses POST / PATCH / DELETE with JSON, every read is a GET, and
// Authorization is the only custom request header the SPA sends.
func corsMiddleware(cfg CORSConfig, next http.Handler) http.Handler {
	if !cfg.enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if echo, ok := cfg.originAllowed(origin); ok {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", echo)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Last-Event-ID")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			// Cap the preflight cache at 10 minutes so config
			// changes to the allow-list propagate quickly without
			// forcing a browser restart.
			h.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			// Preflight: respond 204 regardless of whether the
			// origin matched; an unmatched origin already has no
			// CORS headers attached above, so the browser will
			// reject the actual request anyway.
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
