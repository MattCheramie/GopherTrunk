package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestE4000_TypeAndIF(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	if e.Type() != TypeE4000 {
		t.Errorf("Type() = %v, want E4000", e.Type())
	}
	// Zero-IF tuner — IF freq must be 0.
	if e.IFFreqHz() != 0 {
		t.Errorf("IFFreqHz() = %d, want 0 (zero-IF tuner)", e.IFFreqHz())
	}
}

func TestE4000_GainsLadder(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	g := e.Gains()
	if len(g) != len(e4kLNAGains) {
		t.Errorf("Gains() returned %d, want %d", len(g), len(e4kLNAGains))
	}
}

func TestE4000_SetFreqRangeGuard(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	e.initDone = true
	var rangeErr *ErrUnsupportedFreq
	if err := e.SetFreq(20_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("below-floor err = %v, want *ErrUnsupportedFreq", err)
	}
	if err := e.SetFreq(3_000_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("above-ceiling err = %v, want *ErrUnsupportedFreq", err)
	}
}

func TestE4000PLLRangeTable_BandPicks(t *testing.T) {
	// Spot-check that the band walk picks a row whose divider is
	// non-zero for representative frequencies across the supported
	// range (50 MHz .. 2.2 GHz).
	for _, hz := range []uint32{60_000_000, 100_000_000, 433_000_000, 868_000_000, 1_500_000_000, 2_100_000_000} {
		rng := e4kPLLRanges[len(e4kPLLRanges)-1]
		for _, r := range e4kPLLRanges {
			if hz <= r.freqMax {
				rng = r
				break
			}
		}
		if rng.divLow == 0 {
			t.Errorf("PLL range for %d Hz has zero divider", hz)
		}
	}
}
