package purego

import "github.com/MattCheramie/GopherTrunk/internal/sdr"

// init registers the pure-Go RTL-SDR driver with the global SDR
// registry under the canonical name "rtlsdr". This is now the only
// RTL-SDR backend the project ships — PR-09 removed the legacy CGO
// librtlsdr wrapper along with every `librtlsdr` apt / MSYS2 /
// DLL-bundling step in the build system.
func init() { sdr.Register(&Driver{}) }
