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

// TestE4000_GainsLadder pins the user-facing combined gain ladder against
// librtlsdr's e4k_gains[] (rtlsdr.c) — the list the managed-gain controller
// quantizes against.
func TestE4000_GainsLadder(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	g := e.Gains()
	want := []int{-10, 15, 40, 65, 90, 115, 140, 165, 190, 215, 240, 290, 340, 420}
	if len(g) != len(want) {
		t.Fatalf("Gains() returned %d, want %d", len(g), len(want))
	}
	for i := range want {
		if g[i] != want[i] {
			t.Errorf("Gains()[%d] = %d, want %d (must match librtlsdr e4k_gains[])", i, g[i], want[i])
		}
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

// TestE4000PLLSynthMath pins the (Z, X) Σ-Δ outputs for hand-computed
// frequencies against fosc = 28.8 MHz and the band table. A regression in
// the math or band-table ordering flips a byte downstream of reg 0x09 (Z),
// 0x0A/0x0B (X low/high), 0x0D (band/R).
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
		{"min_boundary_50MHz", 50_000_000, 48, 0x0F, 83, 21_845},
		{"FM_100MHz", 100_000_000, 32, 0x0D, 111, 7281},
		{"ISM_433MHz", 433_000_000, 6, 0x02, 90, 13_653},
		{"ISM_868MHz", 868_000_000, 4, 0x01, 120, 36_408},
		{"L_band_1500MHz", 1_500_000_000, 2, 0x00, 104, 10_922},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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
			if z >= 1<<8 {
				t.Errorf("Z = %d overflows the 8-bit synth field", z)
			}
			if x >= 1<<16 {
				t.Errorf("X = %d overflows the 16-bit fractional field", x)
			}
		})
	}
}

func TestE4000SetFreqBoundaryInclusivity(t *testing.T) {
	e := NewE4000(rtl2832u.New(usb.NewMockTransport()))
	e.initDone = true
	cases := []struct {
		hz        uint32
		wantRange bool
	}{
		{49_999_999, true},
		{50_000_000, false},
		{2_200_000_000, false},
		{2_200_000_001, true},
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

// e4kExpectRead is one scripted E4000 register read returning replyByte.
func e4kExpectRead(reg, replyByte byte) []usb.CtrlExchange {
	return expectI2CReadRegRaw(e4kI2CAddr, reg, replyByte)
}

// e4kExpectWrite is one scripted E4000 register write of {reg, val}.
func e4kExpectWrite(reg, val byte) usb.CtrlExchange {
	return expectI2CWriteRaw(e4kI2CAddr, []byte{reg, val})
}

// TestE4000_SetFreq_WireBytes_VHF pins the SetFreq wire sequence for a VHF2
// carrier (100 MHz) against librtlsdr's e4k_tune_freq → e4k_tune_params:
// SYNTH7 (band/R) first, then Z, then X low/high (e4k_tune_params order),
// then the band bias (BIAS=3 for VHF), then the SYNTH1 band bits and the
// FILT1 RF filter. At 100 MHz the band bits and RF-filter index are 0, so
// with the SYNTH1/FILT1 reads returning 0 those masked writes are skipped.
//
//	100 MHz → row {div=32, bandSel=0x0D}, Z=111 (0x6F), X=7281 (0x1C71)
func TestE4000_SetFreq_WireBytes_VHF(t *testing.T) {
	var script []usb.CtrlExchange
	script = append(script, expectRepeaterToggle(true)...)
	script = append(script,
		e4kExpectWrite(e4kRegSynth7, 0x0D), // R + phase
		e4kExpectWrite(e4kRegSynth3, 0x6F), // Z
		e4kExpectWrite(e4kRegSynth4, 0x71), // X low
		e4kExpectWrite(e4kRegSynth5, 0x1C), // X high
		e4kExpectWrite(e4kRegBias, 0x03),   // band_set: VHF bias
	)
	script = append(script, e4kExpectRead(e4kRegSynth1, 0x00)...) // gap-workaround reset → skip
	script = append(script, e4kExpectRead(e4kRegSynth1, 0x00)...) // band bits 0 → skip
	script = append(script, e4kExpectRead(e4kRegFilt1, 0x00)...)  // RF filter idx 0 → skip
	script = append(script, expectRepeaterToggle(false)...)

	m := usb.NewMockTransport()
	m.Script = script
	e := NewE4000(rtl2832u.New(m))
	e.initDone = true
	if err := e.SetFreq(100_000_000); err != nil {
		t.Fatalf("SetFreq: %v", err)
	}
	if m.Err != nil {
		t.Fatalf("mock: %v", m.Err)
	}
	if m.Step != len(script) {
		t.Errorf("consumed %d/%d exchanges", m.Step, len(script))
	}
}

// TestE4000_SetFreq_WireBytes_UHF pins the band + RF-filter programming for
// a UHF carrier (774.831 MHz — the Ohio MARCS-IP P25 control channel), which
// the pre-fix driver never emitted at all. Reference behaviour:
//
//	774.831 MHz → row {div=4, bandSel=0x01}, Z=107 (0x6B), X=40331 (0x9D8B),
//	flo ≈ 774.83 MHz → band UHF: BIAS=3, SYNTH1 band bits = UHF<<1 = 0x04,
//	FILT1 RF filter = closest UHF centre (760 MHz) = index 12 (0x0C).
//
// The SYNTH1/FILT1 reads return 0 so the masked writes are forced (0 != target).
func TestE4000_SetFreq_WireBytes_UHF(t *testing.T) {
	var script []usb.CtrlExchange
	script = append(script, expectRepeaterToggle(true)...)
	script = append(script,
		e4kExpectWrite(e4kRegSynth7, 0x01),
		e4kExpectWrite(e4kRegSynth3, 0x6B),
		e4kExpectWrite(e4kRegSynth4, 0x8B),
		e4kExpectWrite(e4kRegSynth5, 0x9D),
		e4kExpectWrite(e4kRegBias, 0x03),
	)
	script = append(script, e4kExpectRead(e4kRegSynth1, 0x00)...) // reset: 0&0x06==0 → skip
	script = append(script, e4kExpectRead(e4kRegSynth1, 0x00)...) // band<<1=0x04 → forced
	script = append(script, e4kExpectWrite(e4kRegSynth1, 0x04))
	script = append(script, e4kExpectRead(e4kRegFilt1, 0x00)...) // filter idx 12 → forced
	script = append(script, e4kExpectWrite(e4kRegFilt1, 0x0C))
	script = append(script, expectRepeaterToggle(false)...)

	m := usb.NewMockTransport()
	m.Script = script
	e := NewE4000(rtl2832u.New(m))
	e.initDone = true
	if err := e.SetFreq(774_831_000); err != nil {
		t.Fatalf("SetFreq: %v", err)
	}
	if m.Err != nil {
		t.Fatalf("mock: %v", m.Err)
	}
	if m.Step != len(script) {
		t.Errorf("consumed %d/%d exchanges", m.Step, len(script))
	}
}

// TestE4000_Init_WireBytes pins the full power-on register sequence against
// librtlsdr's e4k_init(), write-for-write. The headline fixes this guards:
// MASTER1 = 0x07 (was 0xC0), the CLKOUT_PWDN = 0x96 write (was missing), the
// IF gain stages and IF filter bandwidths, and the auto-gain AGC setup. All
// read-modify-write reads return 0x00, which makes every masked set with a
// zero target a skip and every non-zero target a forced write.
func TestE4000_Init_WireBytes(t *testing.T) {
	var s []usb.CtrlExchange
	s = append(s, expectRepeaterToggle(true)...)
	s = append(s, e4kExpectRead(0x00, 0x00)...) // dummy read
	s = append(s,
		e4kExpectWrite(e4kRegMaster1, 0x07),    // RESET|NORM_STBY|POR_DET
		e4kExpectWrite(e4kRegClkInp, 0x00),
		e4kExpectWrite(e4kRegRefClk, 0x00),
		e4kExpectWrite(e4kRegClkoutPwdn, 0x96), // disable clock output
		// magic_init
		e4kExpectWrite(0x7E, 0x01), e4kExpectWrite(0x7F, 0xFE),
		e4kExpectWrite(0x82, 0x00), e4kExpectWrite(0x86, 0x50),
		e4kExpectWrite(0x87, 0x20), e4kExpectWrite(0x88, 0x01),
		e4kExpectWrite(0x9F, 0x7F), e4kExpectWrite(0xA0, 0x07),
		// AGC thresholds
		e4kExpectWrite(e4kRegAGC4, 0x10),
		e4kExpectWrite(e4kRegAGC5, 0x04),
		e4kExpectWrite(e4kRegAGC6, 0x1A),
	)
	// enableManualGain(false): AGC1←0x09, AGC7←0x01, AGC11 masked 0 → skip
	s = append(s, e4kExpectRead(e4kRegAGC1, 0x00)...)
	s = append(s, e4kExpectWrite(e4kRegAGC1, 0x09))
	s = append(s, e4kExpectRead(e4kRegAGC7, 0x00)...)
	s = append(s, e4kExpectWrite(e4kRegAGC7, 0x01))
	s = append(s, e4kExpectRead(e4kRegAGC11, 0x00)...) // target 0 → skip
	// IF gain stages 1..6 = 6,0,0,0,9,9
	s = append(s, e4kExpectRead(e4kRegGain3, 0x00)...) // stage1 → 0x01
	s = append(s, e4kExpectWrite(e4kRegGain3, 0x01))
	s = append(s, e4kExpectRead(e4kRegGain3, 0x00)...) // stage2 → skip
	s = append(s, e4kExpectRead(e4kRegGain3, 0x00)...) // stage3 → skip
	s = append(s, e4kExpectRead(e4kRegGain3, 0x00)...) // stage4 → skip
	s = append(s, e4kExpectRead(e4kRegGain4, 0x00)...) // stage5 → 0x02
	s = append(s, e4kExpectWrite(e4kRegGain4, 0x02))
	s = append(s, e4kExpectRead(e4kRegGain4, 0x00)...) // stage6 → 0x10
	s = append(s, e4kExpectWrite(e4kRegGain4, 0x10))
	// IF filters: MIX→FILT2 hi=0xF0, RC→FILT2 lo=0x0F, CHAN→FILT3=0x1F
	s = append(s, e4kExpectRead(e4kRegFilt2, 0x00)...)
	s = append(s, e4kExpectWrite(e4kRegFilt2, 0xF0))
	s = append(s, e4kExpectRead(e4kRegFilt2, 0x00)...)
	s = append(s, e4kExpectWrite(e4kRegFilt2, 0x0F))
	s = append(s, e4kExpectRead(e4kRegFilt3, 0x00)...)
	s = append(s, e4kExpectWrite(e4kRegFilt3, 0x1F))
	s = append(s, e4kExpectRead(e4kRegFilt3, 0x00)...) // chan enable: 0&0x20==0 → skip
	// DC correction disable (all targets 0 → skip)
	s = append(s, e4kExpectRead(e4kRegDC5, 0x00)...)
	s = append(s, e4kExpectRead(e4kRegDCTime1, 0x00)...)
	s = append(s, e4kExpectRead(e4kRegDCTime2, 0x00)...)
	s = append(s, expectRepeaterToggle(false)...)

	m := usb.NewMockTransport()
	m.Script = s
	e := NewE4000(rtl2832u.New(m))
	if err := e.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if m.Err != nil {
		t.Fatalf("mock: %v", m.Err)
	}
	if m.Step != len(s) {
		t.Errorf("consumed %d/%d exchanges", m.Step, len(s))
	}
	if !e.initDone {
		t.Error("initDone not set after Init")
	}
}

// TestE4000_LNAGainReg pins the LNA gain → GAIN1 register mapping against
// librtlsdr's lnagain[]. The register codes are deliberately non-contiguous
// (they skip 2 and 3): the pre-fix table's linear 0x00..0x0F mapping wrote
// the wrong physical gain for most requests.
func TestE4000_LNAGainReg(t *testing.T) {
	cases := []struct {
		tenthDB int
		wantReg byte
	}{
		{-50, 0}, {-25, 1}, {0, 4}, {25, 5}, {50, 6}, {75, 7},
		{100, 8}, {125, 9}, {150, 10}, {175, 11}, {200, 12}, {250, 13}, {300, 14},
		{-1000, 0}, // clamps low
		{9999, 14}, // clamps high
		{10, 4},    // nearest 0 (reg 4) over 25 (reg 5)
		{13, 5},    // nearest 25 (reg 5)
	}
	for _, c := range cases {
		if got := e4kLNAGainReg(c.tenthDB); got != c.wantReg {
			t.Errorf("e4kLNAGainReg(%d) = %d, want %d", c.tenthDB, got, c.wantReg)
		}
	}
}

// TestE4000_MixerGainBit pins the GAIN2 mixer bit: 4 dB → 0, 12 dB → 1
// (librtlsdr e4k_mixer_gain_set).
func TestE4000_MixerGainBit(t *testing.T) {
	if e4kMixerGainBit(4) != 0 {
		t.Error("mixer 4 dB should map to bit 0")
	}
	if e4kMixerGainBit(12) != 1 {
		t.Error("mixer 12 dB should map to bit 1")
	}
}

// TestE4000_BandFor pins the LO → band boundaries (e4k_tune_params).
func TestE4000_BandFor(t *testing.T) {
	cases := []struct {
		flo  uint32
		want e4kBand
	}{
		{100_000_000, e4kBandVHF2},
		{139_999_999, e4kBandVHF2},
		{140_000_000, e4kBandVHF3},
		{349_999_999, e4kBandVHF3},
		{350_000_000, e4kBandUHF},
		{774_831_000, e4kBandUHF},
		{1_134_999_999, e4kBandUHF},
		{1_135_000_000, e4kBandL},
		{1_800_000_000, e4kBandL},
	}
	for _, c := range cases {
		if got := e4kBandFor(c.flo); got != c.want {
			t.Errorf("e4kBandFor(%d) = %d, want %d", c.flo, got, c.want)
		}
	}
}

// TestE4000_ChooseRFFilter pins the RF tracking-filter index against
// librtlsdr's choose_rf_filter / closest_arr_idx tables. VHF uses index 0.
func TestE4000_ChooseRFFilter(t *testing.T) {
	if got := e4kChooseRFFilter(e4kBandVHF2, 100_000_000); got != 0 {
		t.Errorf("VHF2 filter = %d, want 0", got)
	}
	if got := e4kChooseRFFilter(e4kBandVHF3, 200_000_000); got != 0 {
		t.Errorf("VHF3 filter = %d, want 0", got)
	}
	// 774.83 MHz: nearest UHF centre is 760 MHz = index 12.
	if got := e4kChooseRFFilter(e4kBandUHF, 774_830_895); got != 12 {
		t.Errorf("UHF filter @774.83MHz = %d, want 12 (760 MHz tap)", got)
	}
	// 360 MHz: exact first UHF centre = index 0.
	if got := e4kChooseRFFilter(e4kBandUHF, 360_000_000); got != 0 {
		t.Errorf("UHF filter @360MHz = %d, want 0", got)
	}
	// 1300 MHz: exact first L centre = index 0.
	if got := e4kChooseRFFilter(e4kBandL, 1_300_000_000); got != 0 {
		t.Errorf("L filter @1300MHz = %d, want 0", got)
	}
}

// TestE4000_SetGain_Distribution pins the combined-gain LNA+mixer split at
// the top of the ladder (420 = 42 dB): mixer 12 dB (GAIN2 bit 1), LNA capped
// at 30 dB (GAIN1 reg 14 = 0x0E). Matches librtlsdr's e4000_set_gain.
func TestE4000_SetGain_Distribution(t *testing.T) {
	var script []usb.CtrlExchange
	script = append(script, expectRepeaterToggle(true)...)
	script = append(script, e4kExpectRead(e4kRegGain1, 0x00)...) // LNA
	script = append(script, e4kExpectWrite(e4kRegGain1, 0x0E))   // reg for 30 dB
	script = append(script, e4kExpectRead(e4kRegGain2, 0x00)...) // mixer
	script = append(script, e4kExpectWrite(e4kRegGain2, 0x01))   // 12 dB
	script = append(script, expectRepeaterToggle(false)...)

	m := usb.NewMockTransport()
	m.Script = script
	e := NewE4000(rtl2832u.New(m))
	e.initDone = true
	e.manual = true
	if err := e.SetGain(420); err != nil {
		t.Fatalf("SetGain: %v", err)
	}
	if m.Err != nil {
		t.Fatalf("mock: %v", m.Err)
	}
	if m.Step != len(script) {
		t.Errorf("consumed %d/%d exchanges", m.Step, len(script))
	}
}
