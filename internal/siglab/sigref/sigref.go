// Package sigref is SigLab's offline signal-identification reference database —
// a small, curated catalog (Artemis / sigidwiki in spirit) that names a carrier
// by its band, bandwidth, modulation, and symbol rate even when no decoder
// locks. It is the fallback namer for identify and the wideband survey: a
// non-decoding carrier still gets ranked candidates ("looks like TETRA: 25 kHz,
// π/4-DQPSK, 18000 sym/s") rather than vanishing.
//
// The rows for protocols GopherTrunk decodes derive their symbol rate directly
// from the receiver packages' SymbolRate constants, so a constant change
// propagates here at compile time (a drift-guard test pins the rest).
package sigref

import (
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/blind"

	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	dpmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dpmr/receiver"
	dstarrx "github.com/MattCheramie/GopherTrunk/internal/radio/dstar/receiver"
	edacsrx "github.com/MattCheramie/GopherTrunk/internal/radio/edacs/receiver"
	ltrrx "github.com/MattCheramie/GopherTrunk/internal/radio/ltr/receiver"
	motorx "github.com/MattCheramie/GopherTrunk/internal/radio/motorola/receiver"
	mptrx "github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327/receiver"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	ysfrx "github.com/MattCheramie/GopherTrunk/internal/radio/ysf/receiver"
)

// Band is an inclusive frequency range (Hz) a signal is typically allocated in.
type Band struct {
	LoHz, HiHz float64
}

// Entry is one reference signal. For decodable protocols Protocol is the
// trunking protocol string (the identify candidate name) and Decodable is true;
// for catalog-only entries Protocol is "" and Decodable is false.
type Entry struct {
	Protocol     string   `json:"protocol"`
	DisplayName  string   `json:"display_name"`
	ModClass     string   `json:"mod_class"` // one of blind.Mod*
	ModLabels    []string `json:"mod_labels"`
	SymbolRateHz float64  `json:"symbol_rate_hz"`
	ChannelBWHz  float64  `json:"channel_bw_hz"`
	Levels       int      `json:"levels"`
	Bands        []Band   `json:"bands,omitempty"`
	Decodable    bool     `json:"decodable"`
	Notes        string   `json:"notes,omitempty"`
}

// entries is the reference catalog. Decodable rows pull SymbolRateHz from the
// receiver consts so the two cannot drift.
var entries = []Entry{
	{"p25", "P25 Phase 1", blind.ModFSK4, []string{"C4FM"}, p25p1rx.SymbolRate, 12500, 4, nil, true, ""},
	{"p25-phase2", "P25 Phase 2", blind.ModPSK, []string{"H-DQPSK", "TDMA"}, p25p2rx.SymbolRate, 12500, 4, nil, true, ""},
	{"dmr", "DMR", blind.ModFSK4, []string{"4FSK", "TDMA"}, dmrrx.SymbolRate, 12500, 4, nil, true, ""},
	{"nxdn", "NXDN", blind.ModFSK4, []string{"4FSK"}, nxdnrx.SymbolRate, 6250, 4, nil, true, ""},
	{"dpmr", "dPMR", blind.ModFSK4, []string{"4FSK"}, dpmrrx.SymbolRate, 6250, 4, nil, true, ""},
	{"edacs", "EDACS", blind.ModFSK2, []string{"GFSK"}, edacsrx.SymbolRate, 12500, 2, nil, true, ""},
	{"motorola", "Motorola Type II", blind.ModFSK2, []string{"GFSK"}, motorx.SymbolRate, 12500, 2, nil, true, ""},
	{"ltr", "LTR", blind.ModFSK2, []string{"sub-audible FSK"}, ltrrx.SymbolRate, 12500, 2, nil, true, ""},
	{"mpt1327", "MPT-1327", blind.ModFSK2, []string{"FFSK"}, mptrx.SymbolRate, 12500, 2, nil, true, ""},
	{"tetra", "TETRA", blind.ModPSK, []string{"π/4-DQPSK"}, tetrarx.SymbolRate, 25000, 4, nil, true, ""},
	{"ysf", "Yaesu System Fusion", blind.ModFSK4, []string{"C4FM"}, ysfrx.SymbolRate, 12500, 4, nil, true, ""},
	{"dstar", "D-STAR", blind.ModFSK2, []string{"GMSK"}, dstarrx.SymbolRate, 6250, 2, nil, true, ""},

	// Catalog-only reference signals (no GopherTrunk decoder). Kept small.
	{"", "POCSAG-1200", blind.ModFSK2, []string{"2FSK paging"}, 1200, 12500, 2, nil, false, "pager"},
	{"", "POCSAG-512", blind.ModFSK2, []string{"2FSK paging"}, 512, 12500, 2, nil, false, "pager"},
	{"", "FLEX", blind.ModFSK4, []string{"4FSK paging"}, 1600, 25000, 4, nil, false, "pager"},
	{"", "ACARS", blind.ModFSK2, []string{"MSK"}, 2400, 25000, 2, nil, false, "aviation data"},

	// Wideband, non-LMR catalog entries (no decoder, no measurable symbol
	// rate). These name a wideband occupancy span the full-spectrum survey
	// finds — cellular/WiFi/OFDM blocks far wider than any channel. They are
	// ranked by bandwidth + frequency allocation, not symbol rate, and are
	// always reported decodable=false: GopherTrunk identifies and captures
	// them, it does not demodulate them.
	{"", "LTE/5G NR (5 MHz)", blind.ModQAM, []string{"OFDM", "cellular"}, 0, 5_000_000, 0, cellularBands, false, "cellular"},
	{"", "LTE/5G NR (10 MHz)", blind.ModQAM, []string{"OFDM", "cellular"}, 0, 10_000_000, 0, cellularBands, false, "cellular"},
	{"", "LTE/5G NR (15 MHz)", blind.ModQAM, []string{"OFDM", "cellular"}, 0, 15_000_000, 0, cellularBands, false, "cellular"},
	{"", "LTE/5G NR (20 MHz)", blind.ModQAM, []string{"OFDM", "cellular"}, 0, 20_000_000, 0, cellularBands, false, "cellular"},
	{"", "WiFi 802.11 (20 MHz)", blind.ModQAM, []string{"OFDM", "802.11"}, 0, 20_000_000, 0, wifiBands, false, "WLAN"},
	{"", "WiFi 802.11 (40 MHz)", blind.ModQAM, []string{"OFDM", "802.11"}, 0, 40_000_000, 0, wifiBands, false, "WLAN"},
	{"", "WiFi 802.11 (80 MHz)", blind.ModQAM, []string{"OFDM", "802.11"}, 0, 80_000_000, 0, wifiBands, false, "WLAN"},
	{"", "Bluetooth / BLE", blind.ModFSK2, []string{"GFSK", "FHSS"}, 0, 1_000_000, 2, ismBands24, false, "PAN"},
	{"", "ADS-B 1090ES", blind.ModFSK2, []string{"PPM"}, 0, 50_000, 2, adsbBands, false, "aviation"},
}

// Frequency-allocation band sets for the wideband catalog entries. They scope
// the bandwidth+allocation match in Rank so a 10 MHz OFDM block at 751 MHz
// reads as cellular, not WiFi. Edges are approximate (US), for naming only.
var (
	cellularBands = []Band{
		{617e6, 698e6}, {698e6, 806e6}, {824e6, 894e6},
		{1710e6, 1780e6}, {1850e6, 1995e6}, {2110e6, 2200e6},
		{2496e6, 2690e6}, {3550e6, 3700e6},
	}
	wifiBands  = []Band{{2400e6, 2483.5e6}, {5150e6, 5895e6}}
	ismBands24 = []Band{{2400e6, 2483.5e6}}
	adsbBands  = []Band{{1089.9e6, 1090.1e6}, {978e6, 978e6}}
)

// All returns the full catalog (decodable + reference-only).
func All() []Entry { return entries }

// Decodable returns only the entries GopherTrunk has a decoder for.
func Decodable() []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Decodable {
			out = append(out, e)
		}
	}
	return out
}

// ByProtocol returns the decodable entry for a protocol string.
func ByProtocol(p string) (Entry, bool) {
	for _, e := range entries {
		if e.Decodable && e.Protocol == p {
			return e, true
		}
	}
	return Entry{}, false
}

// Observation is what a blind analysis knows about an unknown carrier. Missing
// fields (zero) are simply not scored, so a sparse observation does not penalise.
type Observation struct {
	SymbolRateHz   float64
	SymbolRateConf float64
	ModClass       string
	Levels         int
	ChannelBWHz    float64
	CenterFreqHz   uint32
}

// Match is a scored reference candidate.
type Match struct {
	Entry Entry   `json:"entry"`
	Score float64 `json:"score"` // 0..1
	Why   string  `json:"why"`
}

// Rank scores every catalog entry against obs and returns the top `limit`
// matches by descending score (limit<=0 returns all). The symbol rate carries
// the most weight; modulation class, level count and bandwidth refine it; an
// in-band centre frequency adds a small bonus.
func Rank(obs Observation, limit int) []Match {
	out := make([]Match, 0, len(entries))
	for _, e := range entries {
		s := score(obs, e)
		if s <= 0 {
			continue
		}
		out = append(out, Match{Entry: e, Score: s, Why: why(e)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// widebandMinMatchBwHz is the smallest observed bandwidth that may match a
// wideband catalog entry. It sits far above any land-mobile channel (≤25 kHz)
// so an OFDM/cellular namer can never latch onto a narrowband carrier or a
// low-information blob, while still admitting a ~1 MHz Bluetooth channel.
const widebandMinMatchBwHz = 500_000

func score(obs Observation, e Entry) float64 {
	// Wideband catalog entries (no symbol rate, allocation-scoped) match ONLY a
	// genuinely wideband observation sitting inside the entry's allocation. The
	// allocation is a hard gate, not a soft bonus: an LTE namer fires in a
	// cellular band, a WiFi namer in an ISM band, and neither fires for a
	// narrowband carrier or an out-of-band signal.
	if e.SymbolRateHz == 0 && len(e.Bands) > 0 {
		if obs.ChannelBWHz < widebandMinMatchBwHz || !inAnyBand(obs.CenterFreqHz, e.Bands) {
			return 0
		}
	}

	var acc, totalW float64

	if obs.SymbolRateHz > 0 && e.SymbolRateHz > 0 {
		tol := 0.05 * e.SymbolRateHz
		if tol < 100 {
			tol = 100
		}
		d := obs.SymbolRateHz - e.SymbolRateHz
		if d < 0 {
			d = -d
		}
		sc := 1 - d/tol
		if sc < 0 {
			sc = 0
		}
		conf := obs.SymbolRateConf
		if conf <= 0 {
			conf = 0.5
		}
		acc += 0.5 * sc * conf
		totalW += 0.5
	}

	if obs.ModClass != "" && obs.ModClass != blind.ModUnknown {
		acc += 0.25 * modScore(obs.ModClass, e.ModClass)
		totalW += 0.25
	}

	if obs.Levels > 0 && e.Levels > 0 {
		sc := 0.0
		if obs.Levels == e.Levels {
			sc = 1
		}
		acc += 0.1 * sc
		totalW += 0.1
	}

	if obs.ChannelBWHz > 0 && e.ChannelBWHz > 0 {
		d := obs.ChannelBWHz - e.ChannelBWHz
		if d < 0 {
			d = -d
		}
		sc := 1 - d/e.ChannelBWHz
		if sc < 0 {
			sc = 0
		}
		acc += 0.1 * sc
		totalW += 0.1
	}

	if totalW == 0 {
		return 0
	}
	s := acc / totalW
	if s > 1 {
		s = 1
	}
	return s
}

// inAnyBand reports whether centerHz (Hz) falls inside any of the given bands.
// A zero centre (frequency unknown) is never in-band, so a wideband entry that
// requires an allocation cannot match a frequency-less observation.
func inAnyBand(centerHz uint32, bands []Band) bool {
	if centerHz == 0 {
		return false
	}
	f := float64(centerHz)
	for _, b := range bands {
		if f >= b.LoHz && f <= b.HiHz {
			return true
		}
	}
	return false
}

// modScore is 1 for an exact class match, 0.5 for the same broad family
// (any FSK vs any constant-envelope-PSK vs amplitude), else 0.
func modScore(a, b string) float64 {
	if a == b {
		return 1
	}
	if family(a) == family(b) {
		return 0.5
	}
	return 0
}

func family(m string) string {
	switch m {
	case blind.ModFSK2, blind.ModFSK4:
		return "fsk"
	case blind.ModPSK:
		return "psk"
	case blind.ModQAM:
		return "qam"
	default:
		return m
	}
}

func why(e Entry) string {
	label := ""
	if len(e.ModLabels) > 0 {
		label = e.ModLabels[0]
	}
	return fmt.Sprintf("%.3g kHz, %s, %.0f sym/s", e.ChannelBWHz/1000, label, e.SymbolRateHz)
}
