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
}

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

func score(obs Observation, e Entry) float64 {
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

	// Small bonus for a centre frequency inside one of the entry's bands.
	if obs.CenterFreqHz > 0 {
		f := float64(obs.CenterFreqHz)
		for _, b := range e.Bands {
			if f >= b.LoHz && f <= b.HiHz {
				s += 0.05
				break
			}
		}
	}
	if s > 1 {
		s = 1
	}
	return s
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
