package siglab

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ProtocolDetail is the protocol-specific deep dive for every protocol other
// than P25 Phase 1 (which keeps its richer P25P1Detail with soft eye +
// receiver state). It is computed from the buffered recovered-symbol stream:
// a symbol histogram, a sync-correlation landscape against the protocol's own
// sync word(s), and — where tractable — a per-stage FEC-decode tally. This is
// the symbol-domain analog of P25's FSW landscape + NID-BCH probe, the most
// valuable half of a deep dive, available without any receiver changes.
type ProtocolDetail struct {
	Protocol           string         `json:"protocol" yaml:"protocol"`
	SymbolsBuffered    int            `json:"symbols_buffered" yaml:"symbols_buffered"`
	SymbolCardinality  int            `json:"symbol_cardinality" yaml:"symbol_cardinality"`
	SymbolHistogram    []int64        `json:"symbol_histogram" yaml:"symbol_histogram"`
	SymbolHistogramPct []float64      `json:"symbol_histogram_pct" yaml:"symbol_histogram_pct"`
	Sync               *SyncLandscape `json:"sync,omitempty" yaml:"sync,omitempty"`
	FEC                []FECStat      `json:"fec,omitempty" yaml:"fec,omitempty"`
	Notes              string         `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// FECStat tallies one FEC stage's outcomes across the frames found at clean
// sync hits — the direct analog of P25's NID trusted/marginal/uncorrectable.
type FECStat struct {
	Stage         string `json:"stage" yaml:"stage"`
	Frames        int    `json:"frames" yaml:"frames"`
	Clean         int    `json:"clean" yaml:"clean"`
	Corrected     int    `json:"corrected" yaml:"corrected"`
	Uncorrectable int    `json:"uncorrectable" yaml:"uncorrectable"`
	CRCPass       int    `json:"crc_pass,omitempty" yaml:"crc_pass,omitempty"`
	CRCFail       int    `json:"crc_fail,omitempty" yaml:"crc_fail,omitempty"`
}

// detailSpec describes how to build one protocol's ProtocolDetail: its symbol
// cardinality, sync-correlation tolerance, the sync variant(s) to correlate
// against, an optional FEC tally over the clean sync hits, and notes.
type detailSpec struct {
	cardinality int
	tolerance   int
	syncFn      func() []SyncVariant
	fec         func(stream []uint8, land *SyncLandscape, sys trunking.System) []FECStat
	notes       string
}

// detailSpecs is the per-protocol deep-analysis registry. P25 Phase 1 is
// absent — it uses the dedicated deep path (P25P1Detail).
var detailSpecs = map[trunking.Protocol]detailSpec{
	trunking.ProtocolDMR: {
		cardinality: 4, tolerance: 4, syncFn: dmrSyncVariants, fec: dmrSlotTypeFEC,
	},
	trunking.ProtocolDMRTier2: {
		cardinality: 4, tolerance: 4, syncFn: dmrSyncVariants, fec: dmrSlotTypeFEC,
		notes: "DMR Tier II is voice-only; sync + slot-type only (no CSBK path).",
	},
	trunking.ProtocolDMRTier1: {
		cardinality: 4, tolerance: 4, syncFn: dmrSyncVariants, fec: dmrSlotTypeFEC,
		notes: "DMR Tier I direct-mode; sync + slot-type only (no CSBK path).",
	},
	trunking.ProtocolP25Phase2: {
		cardinality: 4, tolerance: 3,
		syncFn: func() []SyncVariant {
			return []SyncVariant{
				{Name: "outbound", Pattern: p25phase2.OutboundSyncDibits()},
				{Name: "inbound", Pattern: p25phase2.InboundSyncDibits()},
			}
		},
		fec: p25p2FEC,
	},
	trunking.ProtocolNXDN: {
		cardinality: 4, tolerance: 1,
		syncFn: func() []SyncVariant {
			return []SyncVariant{
				{Name: "outbound", Pattern: append([]uint8(nil), nxdn.FSWDibitsOutbound...)},
				{Name: "inbound", Pattern: append([]uint8(nil), nxdn.FSWDibitsInbound...)},
			}
		},
		fec: nxdnFEC,
	},
	trunking.ProtocolDPMR: {
		cardinality: 4, tolerance: 4,
		syncFn: func() []SyncVariant {
			return []SyncVariant{
				{Name: "fs1", Pattern: dpmr.FS1Dibits()},
				{Name: "fs2", Pattern: dpmr.FS2Dibits()},
				{Name: "fs3", Pattern: dpmr.FS3Dibits()},
			}
		},
		notes: "dPMR CSBKs carry no inner FEC; sync landscape only.",
	},
	trunking.ProtocolYSF: {
		cardinality: 4, tolerance: 3,
		syncFn: func() []SyncVariant { return oneVariant("fsw", append([]uint8(nil), ysf.FSWPattern...)) },
		notes:  "FICH decode is not yet integrated in the YSF decoder (FICH-region symbol mapping undefined); sync landscape + histogram only.",
	},
	trunking.ProtocolTETRA: {
		// Training sequences are short (11–19 dibits), so the tolerance
		// is tight to avoid chance correlations; the winner reports its
		// own length via SyncLandscape.SyncLen.
		cardinality: 4, tolerance: 2,
		syncFn: func() []SyncVariant {
			return []SyncVariant{
				{Name: "nts1", Pattern: tetra.NormalSyncDibits()},
				{Name: "nts2", Pattern: tetra.NormalSyncDibits2()},
				{Name: "extended", Pattern: tetra.ExtendedSyncDibits()},
				{Name: "sync", Pattern: tetra.SyncTrainingDibits()},
			}
		},
		fec: tetraFEC,
	},
	trunking.ProtocolEDACS: {
		cardinality: 2, tolerance: 3,
		syncFn: func() []SyncVariant { return oneVariant("outbound", edacs.OutboundSyncBits()) },
		fec:    edacsBCHFEC,
	},
	trunking.ProtocolMotorola: {
		cardinality: 2, tolerance: 3,
		syncFn: func() []SyncVariant { return oneVariant("outbound", motorola.OutboundSyncBits()) },
		fec:    motorolaBCHFEC,
	},
	trunking.ProtocolDStar: {
		cardinality: 2, tolerance: 2,
		syncFn: func() []SyncVariant { return oneVariant("framesync", dstar.FrameSyncBitsSlice()) },
		fec:    dstarCRCFEC,
	},
	trunking.ProtocolMPT1327: {
		cardinality: 2, tolerance: 2,
		syncFn: func() []SyncVariant { return oneVariant("cwsc", mptCWSCBits()) },
		notes:  "MPT codewords run back-to-back; a CWSC need not precede each.",
	},
	trunking.ProtocolLTR: {
		cardinality: 2, tolerance: 0,
		notes: "LTR has no frame-sync word; alignment is Sync-bit based — histogram only.",
	},
}

// hasDetailSpec reports whether a symbol-domain detail builder is registered.
func hasDetailSpec(p trunking.Protocol) bool {
	_, ok := detailSpecs[p]
	return ok
}

// buildProtocolDetail computes the ProtocolDetail for p from the buffered
// symbol stream. Returns nil when no spec is registered or the stream is empty.
func buildProtocolDetail(p trunking.Protocol, symbols []uint8, isBits bool, sys trunking.System) any {
	spec, ok := detailSpecs[p]
	if !ok || len(symbols) == 0 {
		return nil
	}
	card := spec.cardinality
	d := &ProtocolDetail{
		Protocol:          p.String(),
		SymbolsBuffered:   len(symbols),
		SymbolCardinality: card,
		SymbolHistogram:   make([]int64, card),
		Notes:             spec.notes,
	}
	mask := uint8(card - 1)
	for _, s := range symbols {
		d.SymbolHistogram[s&mask]++
	}
	d.SymbolHistogramPct = make([]float64, card)
	for i, n := range d.SymbolHistogram {
		d.SymbolHistogramPct[i] = 100 * float64(n) / float64(len(symbols))
	}

	if spec.syncFn != nil {
		d.Sync = computeSyncLandscape(symbols, spec.syncFn(), card, spec.tolerance)
		if d.Sync != nil && spec.fec != nil {
			d.FEC = spec.fec(symbols, d.Sync, sys)
		}
	}
	return d
}

// dmrSyncVariants converts the 9 ETSI DMR sync words to landscape variants.
func dmrSyncVariants() []SyncVariant {
	out := make([]SyncVariant, 0, len(dmr.AllSyncs))
	for _, s := range dmr.AllSyncs {
		d := s.Dibits
		out = append(out, SyncVariant{Name: s.Name, Pattern: append([]uint8(nil), d[:]...)})
	}
	return out
}

// mptCWSCBits returns the 16-bit MPT 1327 Codeword Synchronisation Code
// (0xC4D7, MSB-first) as a bit pattern.
func mptCWSCBits() []uint8 {
	const cwsc uint16 = 0xC4D7
	out := make([]uint8, 16)
	for i := 0; i < 16; i++ {
		out[i] = uint8((cwsc >> uint(15-i)) & 1)
	}
	return out
}
