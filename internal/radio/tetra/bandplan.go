package tetra

import (
	"errors"
	"fmt"
)

// Resolver maps a TETRA carrier number (the 12-bit Carrier Number
// field of a D-CONNECT PDU) to a frequency in Hz. TETRA networks use
// 25 kHz channel spacing inside a carrier numbering plan that's
// network-specific; both linear and table-backed strategies are
// exposed, mirroring the other protocol packages.
type Resolver interface {
	Frequency(carrier uint16) (uint32, error)
}

// LinearBandPlan applies freq = BaseHz + (carrier + Offset) * SpacingHz.
// Typical values: 380–400 MHz (TETRA-380), 410–430 MHz (TETRA-410),
// or 800–870 MHz (TETRA-800) bases at 25 kHz spacing.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int
}

func (b LinearBandPlan) Frequency(carrier uint16) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, errors.New("tetra/bandplan: SpacingHz must be > 0")
	}
	idx := int(carrier) + b.Offset
	if idx < 0 {
		return 0, fmt.Errorf("tetra/bandplan: carrier %d + offset %d went negative", carrier, b.Offset)
	}
	return b.BaseHz + uint32(idx)*b.SpacingHz, nil
}

// TableBandPlan looks up explicit (carrier → Hz) entries.
type TableBandPlan map[uint16]uint32

func (t TableBandPlan) Frequency(carrier uint16) (uint32, error) {
	hz, ok := t[carrier]
	if !ok {
		return 0, fmt.Errorf("tetra/bandplan: carrier %d not in table", carrier)
	}
	return hz, nil
}
