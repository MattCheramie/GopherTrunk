module github.com/MattCheramie/GopherTrunk

go 1.26.0

// Toolchain pinned to the newest patched release so govulncheck stays
// clean: Go 1.25 is out of support and the October 2026 stdlib advisories
// (GO-2026-6603..6617: net/http HTTP/1 + HTTP/2 desync, HPACK race and flow
// control, Range/MIME header limits, crypto/tls ECH) are fixed only in
// 1.26.9 / 1.27.2. golang.org/x/net v0.60.0 (the same advisories' x/net
// half) requires go 1.26, hence the go directive above. The toolchain
// directive auto-downloads 1.26.9 on a build host running an older Go;
// CI's setup-go is pinned to the same version so the toolchain download
// doesn't run at every CI step.
toolchain go1.26.9

require gopkg.in/yaml.v3 v3.0.1

require (
	github.com/braheezy/shine-mp3 v0.1.0
	github.com/charmbracelet/bubbles v1.0.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/charmbracelet/x/exp/teatest v0.0.0-20260511125431-fe5d686e0c99
	github.com/ebitengine/oto/v3 v3.3.3
	github.com/ebitengine/purego v0.10.0
	github.com/gorilla/websocket v1.5.3
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728
	github.com/mewkiz/flac v1.0.14
	github.com/prometheus/client_golang v1.23.2
	golang.org/x/sys v0.48.0
	golang.org/x/text v0.42.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.11
	modernc.org/sqlite v1.50.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymanbagabas/go-udiff v0.3.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/golden v0.0.0-20241011142426-46044092ad91 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/icza/bitio v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mewkiz/pkg v0.0.0-20250417130911-3f050ff8c56d // indirect
	github.com/mewpkg/term v0.0.0-20241026122259-37a80af23985 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.60.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	modernc.org/libc v1.72.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
