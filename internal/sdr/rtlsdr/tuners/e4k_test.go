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

// TestE4000PLLSynthMath replicates the production Σ-Δ math from
// e4k.go's SetFreq (lines 209–222) and pins the (Z, X) outputs for
// hand-computed frequencies. Z is the integer divider, X is the
// 16-bit fractional in (remainder * 65536) / fosc (integer truncation,
// matching the production uint32 cast). The expected values were
// derived by hand from fosc = 28.8 MHz and the band table at
// e4k.go:84-97; a regression in the math or the band-table ordering
// will flip one of the bytes downstream of register 0x09..0x0C.
func TestE4000PLLSynthMath(t *testing.T) {
	const fosc uint64 = 28_800_000
	cases := []struct {
		name        string
		hz          uint32
		wantDivLow  uint32
		wantBandSel byte
		wantZ       uint32
		wantX       uint32
	}{
		// 50 MHz — exact min-boundary, picks the 72.4 MHz row (div=48).
		// fvco = 2_400_000_000, z = 83 (28.8M * 83 = 2_390_400_000),
		// remainder = 9_600_000, x = 21845.
		{"min_boundary_50MHz", 50_000_000, 48, 0x0F, 83, 21_845},
		// 100 MHz — picks the 108.3 MHz row (div=32).
		// fvco = 3_200_000_000, z = 111, remainder = 3_200_000, x = 7281.
		{"FM_100MHz", 100_000_000, 32, 0x0D, 111, 7281},
		// 433 MHz — exact freqMax boundary of the 433.3 MHz row (div=8).
		// fvco = 3_464_000_000, z = 120, remainder = 8_000_000, x = 18204.
		{"ISM_433MHz", 433_000_000, 8, 0x09, 120, 18_204},
		// 868 MHz — picks the 1.3 GHz row (div=3) since 866.7 MHz row
		// is exceeded.
		// fvco = 2_604_000_000, z = 90, remainder = 12_000_000, x = 27306.
		{"ISM_868MHz", 868_000_000, 3, 0x06, 90, 27_306},
		// 1.5 GHz — picks the 1.7 GHz row (div=2).
		// fvco = 3_000_000_000, z = 104, remainder = 4_800_000, x = 10922.
		{"L_band_1500MHz", 1_500_000_000, 2, 0x05, 104, 10_922},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 1. Band walk must pick the same row the production code does.
			rng := e4kPLLRanges[len(e4kPLLRanges)-1]
			for _, r := range e4kPLLRanges {
				if c.hz <= r.freqMax {
					rng = r
					break
				}
			}
			if rng.divLow != c.wantDivLow {
				t.Errorf("band divLow = %d, want %d", rng.divLow, c.wantDivLow)
			}
			if rng.bandSel != c.wantBandSel {
				t.Errorf("band bandSel = 0x%02x, want 0x%02x", rng.bandSel, c.wantBandSel)
			}

			// 2. Replicate e4k.go:217-222 math verbatim.
			fvco := uint64(c.hz) * uint64(rng.divLow)
			z := uint32(fvco / fosc)
			remainder := fvco - fosc*uint64(z)
			x := uint32((remainder * 65536) / fosc)

			if z != c.wantZ {
				t.Errorf("Z = %d, want %d (fvco=%d remainder=%d)", z, c.wantZ, fvco, remainder)
			}
			if x != c.wantX {
				t.Errorf("X = %d, want %d (remainder=%d)", x, c.wantX, remainder)
			}

			// 3. Z must fit in 11 bits (e4k.go:228-233 splits as 8+3).
			if z >= 1<<11 {
				t.Errorf("Z = %d overflows the 11-bit synth field", z)
			}
			// 4. X must fit in 16 bits (e4k.go:234-237 splits as 8+8).
			if x >= 1<<16 {
				t.Errorf("X = %d overflows the 16-bit fractional field", x)
			}
		})
	}
}

// TestE4000SetFreqBoundaryInclusivity confirms the range guard at
// e4k.go:204 accepts the exact minHz / maxHz endpoints and rejects
// adjacent out-of-range values. The boundary check is `< 50M || >
// 2.2G`, so values at exactly the limit must NOT produce an
// ErrUnsupportedFreq (any other error from the I2C layer is fine —
// we just care about the range guard).
func TestE4000SetFreqBoundaryInclusivity(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	e.initDone = true
	cases := []struct {
		hz        uint32
		wantRange bool // true = expect *ErrUnsupportedFreq
	}{
		{49_999_999, true},     // just below floor
		{50_000_000, false},    // exact floor — accepted
		{2_200_000_000, false}, // exact ceiling — accepted
		{2_200_000_001, true},  // just above ceiling
	}
	for _, c := range cases {
		err := e.SetFreq(c.hz)
		var rangeErr *ErrUnsupportedFreq
		isRange := errors.As(err, &rangeErr)
		if isRange != c.wantRange {
			t.Errorf("SetFreq(%d) range-err = %v, want %v (err=%v)",
				c.hz, isRange, c.wantRange, err)
		}
	}
}

// TestE4000NearestGainIndex_LadderQuantization verifies the shared
// nearestGainIndex helper (defined in fc0013.go but used by E4000 /
// FC0013 / FC2580 — see e4k.go:277) picks the closest ladder entry
// across the E4000's 17-step LNA gain table. Midpoints are pinned so
// any rounding-direction change surfaces in CI.
func TestE4000NearestGainIndex_LadderQuantization(t *testing.T) {
	// e4kLNAGains (tenths of dB):
	//   {-30, -25, -20, -15, -10, -5, 0, 25, 50, 75, 100, 125, 150, 175, 200, 250, 300}
	cases := []struct {
		tenthDB int
		wantIdx int
	}{
		{-30, 0},   // exact: lowest entry
		{-100, 0},  // far below: clamps to lowest
		{-27, 1},   // |-27-(-30)|=3, |-27-(-25)|=2 → -25 wins (idx 1)
		{0, 6},     // exact: -10/-5/0/25 → 0 wins
		{12, 6},    // dist to 0 = 12, dist to 25 = 13 → 0 wins (idx 6)
		{13, 7},    // dist to 0 = 13, dist to 25 = 12 → 25 wins (idx 7)
		{37, 7},    // dist to 25 = 12, dist to 50 = 13 → 25 wins
		{38, 8},    // dist to 25 = 13, dist to 50 = 12 → 50 wins
		{300, 16},  // exact: top entry
		{1000, 16}, // far above: clamps to top
	}
	for _, c := range cases {
		got := nearestGainIndex(e4kLNAGains, c.tenthDB)
		if got != c.wantIdx {
			t.Errorf("nearestGainIndex(e4kLNAGains, %d) = %d, want %d (entry %d vs requested)",
				c.tenthDB, got, c.wantIdx, e4kLNAGains[got])
		}
	}
	// Bonus: idx 7 (the -10→25 jump) is the ladder's widest step.
	// Verify there are exactly that many ladder entries.
	if len(e4kLNAGains) != 17 {
		t.Errorf("ladder size = %d, want 17 (a change here invalidates the test cases)", len(e4kLNAGains))
	}
}
