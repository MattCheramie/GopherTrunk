package ltr

import (
	"errors"
	"fmt"
)

// Resolver maps an LTR Channel number (1..20 typically) to the
// repeater's transmit frequency in Hz. LTR systems use a small
// physical-channel space, often with non-uniform frequency
// assignments — table-based resolution covers most real
// deployments. A linear strategy is also provided for systems that
// happen to lay out their channels uniformly.
type Resolver interface {
	Frequency(channel uint8) (uint32, error)
}

// LinearBandPlan applies freq = BaseHz + (channel + Offset) * SpacingHz.
// LTR conventionally indexes channels from 1, so most operators set
// Offset=-1 to make channel 1 land at BaseHz.
type LinearBandPlan struct {
	BaseHz    uint32
	SpacingHz uint32
	Offset    int // signed
}

func (b LinearBandPlan) Frequency(ch uint8) (uint32, error) {
	if b.SpacingHz == 0 {
		return 0, errors.New("ltr/bandplan: SpacingHz must be > 0")
	}
	idx := int(ch) + b.Offset
	if idx < 0 {
		return 0, fmt.Errorf("ltr/bandplan: channel %d + offset %d went negative", ch, b.Offset)
	}
	return b.BaseHz + uint32(idx)*b.SpacingHz, nil
}

// TableBandPlan looks up explicit (channel → Hz) entries.
type TableBandPlan map[uint8]uint32

func (t TableBandPlan) Frequency(ch uint8) (uint32, error) {
	hz, ok := t[ch]
	if !ok {
		return 0, fmt.Errorf("ltr/bandplan: channel %d not in table", ch)
	}
	return hz, nil
}
