package tier3

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Resolver maps a 12-bit DMR Tier III LCN (Logical Physical Channel
// Number) to its downlink frequency in Hz. Two implementations ship:
// LinearBandPlan for sites that lay channels out on a regular
// base+spacing grid, and TableBandPlan for the irregular cases (where
// the operator hand-codes the LCN → Hz mapping from a license /
// coordination database).
type Resolver interface {
	Frequency(lcn uint16) (uint32, error)
}

// ErrUnknownLCN is returned by a Resolver when the supplied LCN is
// outside the configured plan. The control channel maps it to a
// `decode.error` event with stage="no-bandplan" so operators can spot
// configuration gaps in metrics.
var ErrUnknownLCN = errors.New("dmr/tier3: LCN outside band plan")

// LinearBandPlan resolves LCN → BaseHz + (LCN - Offset) × SpacingHz.
// Offset lets sites that start their LCN numbering at 1 (the most
// common case) keep BaseHz pinned to the actual channel-1 downlink.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int8 // typically 1 to match LCN-1-indexed sites
}

func (b LinearBandPlan) Frequency(lcn uint16) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, fmt.Errorf("dmr/tier3: linear band plan needs non-zero SpacingHz")
	}
	idx := int32(lcn) - int32(b.Offset)
	if idx < 0 {
		return 0, fmt.Errorf("%w: lcn=%d below Offset=%d", ErrUnknownLCN, lcn, b.Offset)
	}
	hz := uint64(b.BaseHz) + uint64(idx)*uint64(b.SpacingHz)
	if hz > 0xFFFFFFFF {
		return 0, fmt.Errorf("dmr/tier3: resolved frequency %d Hz overflows uint32", hz)
	}
	return uint32(hz), nil
}

// TableBandPlan is a hand-coded LCN → Hz lookup. The map is consulted
// directly; missing keys return ErrUnknownLCN.
type TableBandPlan map[uint16]uint32

func (t TableBandPlan) Frequency(lcn uint16) (uint32, error) {
	hz, ok := t[lcn]
	if !ok {
		return 0, fmt.Errorf("%w: lcn=%d", ErrUnknownLCN, lcn)
	}
	return hz, nil
}

// ResolverFromPlan builds a Resolver from the operator-supplied band
// plan carried on a trunking.System. It returns nil when p is nil or
// empty so callers can pass the result straight into Options.Resolver
// and preserve the existing "no band plan ⇒ drop grant with
// stage=no-bandplan" behaviour. Config validation guarantees exactly
// one of Linear / Table is set; if both are somehow present Linear
// wins.
func ResolverFromPlan(p *trunking.DMRBandPlan) Resolver {
	if p == nil {
		return nil
	}
	if p.Linear != nil {
		return LinearBandPlan{
			BaseHz:    p.Linear.BaseHz,
			SpacingHz: p.Linear.SpacingHz,
			Offset:    p.Linear.Offset,
		}
	}
	if len(p.Table) > 0 {
		t := make(TableBandPlan, len(p.Table))
		for _, e := range p.Table {
			t[e.LCN] = e.FreqHz
		}
		return t
	}
	return nil
}
