package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestFC0013_TypeAndIF(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	if f.Type() != TypeFC0013 {
		t.Errorf("Type() = %v, want FC0013", f.Type())
	}
	if f.IFFreqHz() != 6_000_000 {
		t.Errorf("IFFreqHz() = %d, want 6_000_000", f.IFFreqHz())
	}
}

func TestFC0013_GainsLadder26Steps(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	g := f.Gains()
	if len(g) != 26 {
		t.Errorf("Gains() returned %d entries, want 26 (librtlsdr fc0013_lna_gains)", len(g))
	}
	if g[0] != -99 || g[len(g)-1] != 118 {
		t.Errorf("Gains() endpoints = (%d, %d), want (-99, 118)", g[0], g[len(g)-1])
	}
}

func TestFC0013_InitWritesSoftResetThenFlood(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, 0x0C, 0x05)...)
	m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, 0x0C, 0x00)...)
	for i, v := range fc0013InitArray {
		m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, byte(i+1), v)...)
	}
	f := NewFC0013(rtl2832u.New(m))
	if err := f.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestFC0013_SetFreqRangeGuard(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	var rangeErr *ErrUnsupportedFreq
	if err := f.SetFreq(10_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("below-floor err = %v, want *ErrUnsupportedFreq", err)
	}
	if err := f.SetFreq(2_000_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("above-ceiling err = %v, want *ErrUnsupportedFreq", err)
	}
}

// TestFC0013SetFreqBoundaryInclusivity confirms the range guard at
// fc0013.go:110 accepts the exact 22 MHz / 1.7 GHz endpoints.
func TestFC0013SetFreqBoundaryInclusivity(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	cases := []struct {
		hz        uint32
		wantRange bool
	}{
		{21_999_999, true},
		{22_000_000, false},
		{1_700_000_000, false},
		{1_700_000_001, true},
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

// TestNearestGainIndex_SharedHelper verifies the shared
// nearestGainIndex (fc0013.go:268) routes midpoint requests to the
// numerically closer ladder entry. The FC0013 LNA ladder is the
// longest of the three tuners that use the shared helper, so it
// exercises the search loop most.
func TestNearestGainIndex_SharedHelper(t *testing.T) {
	// fc0013LNAGains has 26 entries; spot-check rounding behavior at
	// midpoints + at the clamp boundaries.
	cases := []struct {
		tenthDB int
		check   func(int) bool
		desc    string
	}{
		// Below the ladder floor: must clamp to index 0.
		{tenthDB: -10_000, check: func(i int) bool { return i == 0 }, desc: "clamp low"},
		// Far above any reasonable ladder value: must clamp to len-1.
		{tenthDB: 10_000, check: func(i int) bool { return i == len(fc0013LNAGains)-1 }, desc: "clamp high"},
		// Exact ladder hit picks itself.
		{tenthDB: fc0013LNAGains[5], check: func(i int) bool { return i == 5 }, desc: "exact match idx=5"},
		// A small positive delta picks the nearer of two adjacent
		// entries — assert it's adjacent to the exact match.
		{tenthDB: fc0013LNAGains[10] + 1, check: func(i int) bool { return i == 10 || i == 11 }, desc: "near idx=10"},
	}
	for _, c := range cases {
		got := nearestGainIndex(fc0013LNAGains, c.tenthDB)
		if !c.check(got) {
			t.Errorf("%s: nearestGainIndex(fc0013LNAGains, %d) = %d (entry %d)",
				c.desc, c.tenthDB, got, fc0013LNAGains[got])
		}
	}

	// Empty ladder — production code panics by indexing ladder[0]. We
	// don't test that explicitly because every tuner passes a populated
	// ladder; this comment documents the invariant.

	// Tie-break: when two entries are equidistant, the FIRST one wins
	// (the loop uses `if d < bestDist`, not `<=`). Construct a
	// synthetic ladder to pin that.
	synth := []int{0, 100}
	if got := nearestGainIndex(synth, 50); got != 0 {
		t.Errorf("tie-break: nearestGainIndex({0, 100}, 50) = %d, want 0 (first-wins)", got)
	}
}
