package motorola

import (
	"errors"
	"fmt"
)

// BandPlan resolves a Motorola Logical Channel Number (LCN) to a
// frequency in Hz. Two strategies cover the bulk of real systems:
//
//   - Linear: freq = base + (lcn + offset) * spacing — typical for
//     800 / 900 MHz NPSPAC and many state-wide SmartZone systems.
//   - Table:  explicit per-LCN entries — for VHF / UHF systems whose
//     channel plan doesn't follow a single linear formula.
//
// Use whichever matches the operator's configured system. The control
// state machine takes a `Resolver` interface so live captures can
// switch between them per System without code changes.
type Resolver interface {
	Frequency(lcn uint16) (uint32, error)
}

// LinearBandPlan applies freq = BaseHz + (lcn + Offset) * SpacingHz.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int // signed; supports negative offsets used by some plans
}

func (b LinearBandPlan) Frequency(lcn uint16) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, errors.New("motorola/bandplan: SpacingHz must be > 0")
	}
	idx := int(lcn) + b.Offset
	if idx < 0 {
		return 0, fmt.Errorf("motorola/bandplan: LCN %d + offset %d went negative", lcn, b.Offset)
	}
	return b.BaseHz + uint32(idx)*b.SpacingHz, nil
}

// TableBandPlan looks up explicit (lcn → Hz) entries. Useful for
// VHF / UHF systems with non-linear channel plans.
type TableBandPlan map[uint16]uint32

func (t TableBandPlan) Frequency(lcn uint16) (uint32, error) {
	hz, ok := t[lcn]
	if !ok {
		return 0, fmt.Errorf("motorola/bandplan: LCN %d not in table", lcn)
	}
	return hz, nil
}
