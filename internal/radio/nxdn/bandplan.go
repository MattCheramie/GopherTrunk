package nxdn

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Resolver maps an NXDN traffic-channel number (as carried in a
// VCALL_ASSGN message) to its downlink frequency in Hz. Two
// implementations ship, mirroring the DMR Tier III band plan
// (internal/radio/dmr/tier3): LinearBandPlan for sites that lay
// channels out on a regular base+spacing grid, and TableBandPlan for
// the irregular cases where the operator hand-codes the channel → Hz
// mapping from a license / coordination database.
type Resolver interface {
	Frequency(channel uint16) (uint32, error)
}

// ErrUnknownChannel is returned by a Resolver when the supplied
// channel is outside the configured plan. The control channel drops
// such grants (they cannot be tuned) rather than publishing them.
var ErrUnknownChannel = errors.New("nxdn: channel outside band plan")

// LinearBandPlan resolves channel → BaseHz + (channel - Offset) ×
// SpacingHz. Offset lets sites that start their channel numbering at 1
// keep BaseHz pinned to the actual channel-1 downlink.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int8 // typically 1 to match channel-1-indexed sites
}

func (b LinearBandPlan) Frequency(channel uint16) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, fmt.Errorf("nxdn: linear band plan needs non-zero SpacingHz")
	}
	idx := int32(channel) - int32(b.Offset)
	if idx < 0 {
		return 0, fmt.Errorf("%w: channel=%d below Offset=%d", ErrUnknownChannel, channel, b.Offset)
	}
	hz := uint64(b.BaseHz) + uint64(idx)*uint64(b.SpacingHz)
	if hz > 0xFFFFFFFF {
		return 0, fmt.Errorf("nxdn: resolved frequency %d Hz overflows uint32", hz)
	}
	return uint32(hz), nil
}

// TableBandPlan is a hand-coded channel → Hz lookup. The map is
// consulted directly; missing keys return ErrUnknownChannel.
type TableBandPlan map[uint16]uint32

func (t TableBandPlan) Frequency(channel uint16) (uint32, error) {
	hz, ok := t[channel]
	if !ok {
		return 0, fmt.Errorf("%w: channel=%d", ErrUnknownChannel, channel)
	}
	return hz, nil
}

// ResolverFromPlan builds a Resolver from the operator-supplied band
// plan carried on a trunking.System. It returns nil when p is nil or
// empty so callers can pass the result straight into
// ControlChannel.SetBandPlan and preserve the "no band plan ⇒ drop
// grant" behaviour. Config validation guarantees exactly one of Linear
// / Table is set; if both are somehow present Linear wins.
func ResolverFromPlan(p *trunking.NXDNBandPlan) Resolver {
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
			t[e.Channel] = e.FreqHz
		}
		return t
	}
	return nil
}
