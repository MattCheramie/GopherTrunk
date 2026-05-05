package dpmr

import (
	"errors"
	"fmt"
)

// Resolver maps a dPMR channel number (the 16-bit Extra field of a
// voice-allocation CSBK) to a frequency in Hz. dPMR systems use
// 6.25 kHz channel spacing by default; both linear and table-backed
// strategies are exposed, mirroring the other protocol packages.
type Resolver interface {
	Frequency(channel uint16) (uint32, error)
}

// LinearBandPlan applies freq = BaseHz + (channel + Offset) * SpacingHz.
// For PMR446 the canonical layout is BaseHz = 446_006_250, SpacingHz =
// 6_250, Offset = -1 so channel 1 = 446 006 250 Hz.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int
}

func (b LinearBandPlan) Frequency(ch uint16) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, errors.New("dpmr/bandplan: SpacingHz must be > 0")
	}
	idx := int(ch) + b.Offset
	if idx < 0 {
		return 0, fmt.Errorf("dpmr/bandplan: channel %d + offset %d went negative", ch, b.Offset)
	}
	return b.BaseHz + uint32(idx)*b.SpacingHz, nil
}

// TableBandPlan looks up explicit (channel → Hz) entries.
type TableBandPlan map[uint16]uint32

func (t TableBandPlan) Frequency(ch uint16) (uint32, error) {
	hz, ok := t[ch]
	if !ok {
		return 0, fmt.Errorf("dpmr/bandplan: channel %d not in table", ch)
	}
	return hz, nil
}
