package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestFC2580_TypeAndIF(t *testing.T) {
	f := NewFC2580(rtl2832u.New(usb.NewMockTransport()))
	if f.Type() != TypeFC2580 {
		t.Errorf("Type() = %v, want FC2580", f.Type())
	}
	if f.IFFreqHz() != 5_600_000 {
		t.Errorf("IFFreqHz() = %d, want 5_600_000 (VHF default)", f.IFFreqHz())
	}
}

func TestFC2580_Gains4Steps(t *testing.T) {
	f := NewFC2580(rtl2832u.New(usb.NewMockTransport()))
	g := f.Gains()
	if len(g) != 4 {
		t.Errorf("Gains() returned %d entries, want 4 (coarse ladder)", len(g))
	}
}

func TestFC2580_SetFreqRangeGuard(t *testing.T) {
	f := NewFC2580(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	var rangeErr *ErrUnsupportedFreq
	if err := f.SetFreq(40_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("below-floor err = %v, want *ErrUnsupportedFreq", err)
	}
	if err := f.SetFreq(3_000_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("above-ceiling err = %v, want *ErrUnsupportedFreq", err)
	}
}

// TestFC2580SetFreqBoundaryInclusivity confirms the range guard at
// fc2580.go:134 accepts the exact 50 MHz / 2.6 GHz endpoints.
func TestFC2580SetFreqBoundaryInclusivity(t *testing.T) {
	f := NewFC2580(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	cases := []struct {
		hz        uint32
		wantRange bool
	}{
		{49_999_999, true},
		{50_000_000, false},
		{2_600_000_000, false},
		{2_600_000_001, true},
	}
	for _, c := range cases {
		err := f.SetFreq(c.hz)
		var rangeErr *ErrUnsupportedFreq
		isRange := errors.As(err, &rangeErr)
		if isRange != c.wantRange {
			t.Errorf("SetFreq(%d) range-err = %v, want %v (err=%v)",
				c.hz, isRange, c.wantRange, err)
		}
	}
}

func TestFC2580BandSelect_IFChangesAcrossBands(t *testing.T) {
	// Verify the band table picks different IF frequencies for VHF
	// (5.6 MHz) vs UHF (4.6 MHz).
	cases := []struct {
		hz     uint32
		wantIF uint32
	}{
		{80_000_000, 5_600_000},    // FM VHF-II
		{150_000_000, 5_600_000},   // VHF-III / DAB
		{500_000_000, 4_600_000},   // UHF
		{700_000_000, 4_600_000},   // UHF (above 500)
		{1_500_000_000, 1_400_000}, // L-band fallback
	}
	for _, c := range cases {
		band := fc2580Bands[len(fc2580Bands)-1]
		for _, b := range fc2580Bands {
			if c.hz <= b.freqMax {
				band = b
				break
			}
		}
		if band.ifFreqHz != c.wantIF {
			t.Errorf("band for %d Hz has IF=%d, want %d", c.hz, band.ifFreqHz, c.wantIF)
		}
	}
}
