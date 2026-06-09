package radioreference

import (
	"os"
	"strings"
)

// DefaultAppKey is the built-in RadioReference application key. It is injected
// at BUILD time via the linker, e.g.:
//
//	go build -ldflags "-X github.com/MattCheramie/GopherTrunk/internal/radioreference.DefaultAppKey=<key>"
//
// (the Makefile wires this from the RR_APP_KEY environment variable, and the
// release workflow from the RR_APP_KEY Actions secret). It is intentionally
// empty in a bare `go build` / `go test` / `go run` and is never committed to
// source — a build without the key simply falls back to env/config, exactly as
// before this default existed.
var DefaultAppKey = ""

// ResolveAuth fills an empty AppKey from the GOPHERTRUNK_RR_KEY environment
// variable and then the built-in DefaultAppKey, leaving an explicitly-supplied
// key (and the username/password) untouched. The resulting key precedence is:
// the value the caller already resolved (flag/config) > env > built-in. Every
// place that assembles an Auth should pass it through here so the built-in key
// flows everywhere uniformly.
func ResolveAuth(a Auth) Auth {
	a.AppKey = firstNonEmpty(
		strings.TrimSpace(a.AppKey),
		strings.TrimSpace(os.Getenv("GOPHERTRUNK_RR_KEY")),
		strings.TrimSpace(DefaultAppKey),
	)
	return a
}

// EffectiveAppKey resolves just the app key (explicit > env > built-in) for
// callers that don't have a full Auth in hand.
func EffectiveAppKey(explicit string) string {
	return ResolveAuth(Auth{AppKey: explicit}).AppKey
}
