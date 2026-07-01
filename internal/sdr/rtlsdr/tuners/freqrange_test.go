package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// freqRanger is the per-chip FreqRange contract the whole-device hunt
// sweep relies on. Every Tuner satisfies it.
type freqRanger interface {
	FreqRange() (minHz, maxHz uint32)
	SetFreq(hz uint32) error
}

// TestTunerFreqRange pins each tuner's reported range to its documented
// PLL limits AND proves the SetFreq guard rejects just-outside-range
// requests with an ErrUnsupportedFreq whose Min/Max equal FreqRange().
// Because the guard now reads its bounds from FreqRange(), this is the
// drift guard for the Phase-1 single-source refactor: the advertised
// range and the enforced range can never disagree.
func TestTunerFreqRange(t *testing.T) {
	newDemod := func() *rtl2832u.Demod { return rtl2832u.New(usb.NewMockTransport()) }

	cases := []struct {
		name    string
		tuner   freqRanger
		wantMin uint32
		wantMax uint32
	}{
		{"R820T2", func() *R82xx { r := NewR82xx(newDemod(), r82xxI2CAddr, TypeR820T2); r.initDone = true; return r }(), 24_000_000, 1_766_000_000},
		{"E4000", func() *E4000 { e := NewE4000(newDemod()); e.initDone = true; return e }(), 50_000_000, 2_200_000_000},
		{"FC0012", func() *FC0012 { f := NewFC0012(newDemod()); f.initDone = true; return f }(), 37_000_000, 1_700_000_000},
		{"FC0013", func() *FC0013 { f := NewFC0013(newDemod()); f.initDone = true; return f }(), 22_000_000, 1_700_000_000},
		{"FC2580", func() *FC2580 { f := NewFC2580(newDemod()); f.initDone = true; return f }(), 50_000_000, 2_600_000_000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			min, max := c.tuner.FreqRange()
			if min != c.wantMin || max != c.wantMax {
				t.Fatalf("FreqRange() = (%d, %d), want (%d, %d)", min, max, c.wantMin, c.wantMax)
			}
			if min >= max {
				t.Fatalf("FreqRange() min %d >= max %d", min, max)
			}

			// Below the floor and above the ceiling must both report
			// ErrUnsupportedFreq carrying exactly the advertised bounds.
			for _, hz := range []uint32{min - 1, max + 1} {
				var rangeErr *ErrUnsupportedFreq
				err := c.tuner.SetFreq(hz)
				if !errors.As(err, &rangeErr) {
					t.Fatalf("SetFreq(%d) err = %v, want *ErrUnsupportedFreq", hz, err)
				}
				if rangeErr.MinHz != c.wantMin || rangeErr.MaxHz != c.wantMax {
					t.Errorf("SetFreq(%d) guard bounds = (%d, %d), want (%d, %d)",
						hz, rangeErr.MinHz, rangeErr.MaxHz, c.wantMin, c.wantMax)
				}
			}
		})
	}
}
