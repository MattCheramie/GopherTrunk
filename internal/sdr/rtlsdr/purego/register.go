//go:build rtlsdr_purego

package purego

import "github.com/MattCheramie/GopherTrunk/internal/sdr"

// init registers the pure-Go RTL-SDR driver with the global SDR
// registry. Gated by the rtlsdr_purego build tag so the CGO and
// pure-Go drivers can coexist during the rewrite (PR-08 will swap
// names; PR-09 will remove the CGO wrapper entirely).
//
// Default builds (no -tags rtlsdr_purego) skip this file, leaving
// the package importable but inert — the blank import in
// cmd/gophertrunk/main.go is then a compile-time no-op for SDR
// registration. Existing CGO builds keep registering as "rtlsdr"
// from internal/sdr/rtlsdr/rtlsdr_cgo.go.
func init() { sdr.Register(&Driver{}) }
