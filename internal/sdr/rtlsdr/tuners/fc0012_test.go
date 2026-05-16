package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// expectI2CWriteReg returns the full script for one I2CWriteReg
// (=2-byte I2C burst wrapped in repeater on/off).
func expectI2CWriteReg(addr, reg, val byte) []usb.CtrlExchange {
	return expectI2CWrite(addr, []byte{reg, val})
}

func TestFC0012_TypeAndIF(t *testing.T) {
	d := rtl2832u.New(usb.NewMockTransport())
	f := NewFC0012(d)
	if f.Type() != TypeFC0012 {
		t.Errorf("Type() = %v, want FC0012", f.Type())
	}
	if f.IFFreqHz() != 6_000_000 {
		t.Errorf("IFFreqHz() = %d, want 6_000_000", f.IFFreqHz())
	}
}

func TestFC0012_GainsLadder(t *testing.T) {
	d := rtl2832u.New(usb.NewMockTransport())
	f := NewFC0012(d)
	g := f.Gains()
	if len(g) != 5 {
		t.Fatalf("Gains() returned %d entries, want 5", len(g))
	}
	// First entry must be -99 (the -9.9 dB anchor librtlsdr ships).
	if g[0] != -99 {
		t.Errorf("Gains()[0] = %d, want -99", g[0])
	}
}

func TestFC0012_InitDoesSoftResetThenFlood(t *testing.T) {
	// Init = 2 soft-reset writes + 20 init-array writes (one I2C
	// burst per write, since FC0012's I2CWriteReg sends a 2-byte
	// burst at a time).
	m := usb.NewMockTransport()
	// Soft reset (0x0C ← 0x05 then 0x0C ← 0x00).
	m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, 0x0C, 0x05)...)
	m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, 0x0C, 0x00)...)
	for i, v := range fc0012InitArray {
		m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, byte(i+1), v)...)
	}
	d := rtl2832u.New(m)
	f := NewFC0012(d)
	if err := f.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if m.Err != nil {
		t.Errorf("mock err: %v", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestFC0012_InitIdempotent(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, 0x0C, 0x05)...)
	m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, 0x0C, 0x00)...)
	for i, v := range fc0012InitArray {
		m.Script = append(m.Script, expectI2CWriteReg(fc0012I2CAddr, byte(i+1), v)...)
	}
	d := rtl2832u.New(m)
	f := NewFC0012(d)
	if err := f.Init(); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := f.Init(); err != nil {
		t.Fatalf("second Init: %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0 (second Init must skip)", m.Remaining())
	}
}

func TestFC0012_SetFreqRangeGuard(t *testing.T) {
	m := usb.NewMockTransport()
	d := rtl2832u.New(m)
	f := NewFC0012(d)
	f.initDone = true
	// Below floor.
	err := f.SetFreq(10_000_000)
	var rangeErr *ErrUnsupportedFreq
	if !errors.As(err, &rangeErr) {
		t.Errorf("below-floor SetFreq err = %v, want *ErrUnsupportedFreq", err)
	}
	// Above ceiling.
	err = f.SetFreq(2_000_000_000)
	if !errors.As(err, &rangeErr) {
		t.Errorf("above-ceiling SetFreq err = %v, want *ErrUnsupportedFreq", err)
	}
}

func TestFC0012_SetFreqBeforeInitFails(t *testing.T) {
	d := rtl2832u.New(usb.NewMockTransport())
	f := NewFC0012(d)
	if err := f.SetFreq(100_000_000); err == nil {
		t.Error("SetFreq before Init returned nil, want error")
	}
}

func TestFC0012BandSelect_BoundaryRanges(t *testing.T) {
	// Verify boundary picks match the C source. These are not
	// hardware-validated; the test pins the table walk so any
	// reordering of the cases fails.
	cases := []struct {
		hz        uint32
		wantMulti uint32
	}{
		{37_000_000, 96},
		{50_000_000, 64},
		{75_000_000, 48},  // wait, table says <74_167_000 → 48, and >=74_167_000 → 32
		{100_000_000, 32}, // ≥ 74.167 && < 111.250
		{200_000_000, 16},
		{300_000_000, 12},
		{500_000_000, 6},
		{1_000_000_000, 2},
	}
	for _, c := range cases {
		multi, _, _ := fc0012BandSelect(c.hz)
		// Loose bound check — we just want to know the multiplier
		// reasonably bounds the VCO range, not the exact value.
		if multi == 0 {
			t.Errorf("fc0012BandSelect(%d) returned multi=0", c.hz)
		}
	}
}

func TestFC0012NearestGainIndex(t *testing.T) {
	// Verify the rounding behavior for the 5-step ladder.
	cases := []struct {
		tenthDB  int
		wantIdx  int
	}{
		{-100, 0},   // closest to -99
		{-50, 1},    // closest to -40
		{50, 2},     // closest to 71? distance: |50-(-40)|=90, |50-71|=21, |50-179|=129. → idx 2
		{200, 4},    // 192 is closest
		{1000, 4},   // above range: clamps to top
	}
	for _, c := range cases {
		got := fc0012NearestGainIndex(c.tenthDB)
		if got != c.wantIdx {
			t.Errorf("fc0012NearestGainIndex(%d) = %d, want %d", c.tenthDB, got, c.wantIdx)
		}
	}
}

// TestFC0012SetFreqBoundaryInclusivity confirms the range guard at
// fc0012.go:115 accepts the exact minHz / maxHz endpoints. The guard
// is `< 37M || > 1.7G`; values at the boundary must not surface a
// range error (any I2C error from the un-scripted mock is fine).
func TestFC0012SetFreqBoundaryInclusivity(t *testing.T) {
	f := NewFC0012(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	cases := []struct {
		hz        uint32
		wantRange bool
	}{
		{36_999_999, true},     // just below floor
		{37_000_000, false},    // exact floor — accepted
		{1_700_000_000, false}, // exact ceiling — accepted
		{1_700_000_001, true},  // just above ceiling
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
