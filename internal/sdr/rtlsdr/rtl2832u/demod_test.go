package rtl2832u

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// expectDemodWrite builds the (write, commit-read) script entries for one
// demod register write. Keeps the InitBaseband golden table readable.
func expectDemodWrite(page uint8, addr, val uint16, n int) []usb.CtrlExchange {
	wValue := (addr << 8) | 0x20
	wIndex := uint16(0x10) | uint16(page)
	return []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: wValue, WIndex: wIndex, Data: encodeWriteVal(val, n)},
		commit,
	}
}

func expectBlockWrite(block uint8, addr, val uint16, n int) usb.CtrlExchange {
	return usb.CtrlExchange{
		In:       false,
		BRequest: 0,
		WValue:   addr,
		WIndex:   uint16(block)<<8 | 0x10,
		Data:     encodeWriteVal(val, n),
	}
}

func TestComputeResamplerDivisor_GoldenTable(t *testing.T) {
	// Each row was computed against the published librtlsdr formula
	// with the default 28.8 MHz reference crystal:
	//
	//   rsamp_ratio = (xtal * 2^22) / samp_rate
	//   rsamp_ratio &= 0x0FFFFFFC
	//   real_rsamp_ratio = rsamp_ratio | ((rsamp_ratio & 0x08000000) << 1)
	//   real_rate = (xtal * 2^22) / real_rsamp_ratio
	//
	// The values pin both the divisor written to demod pages 0x9F/0xA1
	// and the post-quantization "real" rate that downstream DSP will
	// see.
	cases := []struct {
		samp       uint32
		wantDiv    uint32
		wantRealHz uint32
	}{
		// 2.4 MS/s — the project's default working rate.
		// xtal*2^22 / 2_400_000 = 50_331_648 = 0x03000000.
		{samp: 2_400_000, wantDiv: 0x03000000, wantRealHz: 2_400_000},
		// 1.024 MS/s. xtal*2^22 / 1_024_000 = 117_964_800 = 0x07080000.
		{samp: 1_024_000, wantDiv: 0x07080000, wantRealHz: 1_024_000},
		// 2.048 MS/s. xtal*2^22 / 2_048_000 = 58_982_400 = 0x03840000.
		{samp: 2_048_000, wantDiv: 0x03840000, wantRealHz: 2_048_000},
		// 3.2 MS/s — the upper limit librtlsdr admits.
		// xtal*2^22 / 3_200_000 = 37_748_736 = 0x02400000.
		{samp: 3_200_000, wantDiv: 0x02400000, wantRealHz: 3_200_000},
		// 250 kS/s — below the 300 kHz forbidden floor; hits the
		// sign-bit expansion path. Pre-mask ratio is 0x1CCCCCCC;
		// the mask clears bits 28+, leaving 0x0CCCCCCC. The
		// sign-extend OR then puts bit 27 back into the realRatio
		// used for the realHz computation, but the divisor written
		// to demod 0x9F/0xA1 stays 0x0CCCCCCC.
		{samp: 250_000, wantDiv: 0x0CCCCCCC, wantRealHz: 250_000},
	}
	for _, tc := range cases {
		div, realHz := computeResamplerDivisor(DefaultXtalHz, tc.samp)
		if div != tc.wantDiv {
			t.Errorf("computeResamplerDivisor(%d) divisor = 0x%08X, want 0x%08X", tc.samp, div, tc.wantDiv)
		}
		// Real-rate quantization: assert we're within 1 Hz of the
		// requested rate (the divisor has 28-bit resolution so the
		// quantization step at 2.4 MS/s is ~0.05 Hz).
		diff := int64(realHz) - int64(tc.wantRealHz)
		if diff < -1 || diff > 1 {
			t.Errorf("computeResamplerDivisor(%d) realHz = %d, want %d (±1)", tc.samp, realHz, tc.wantRealHz)
		}
	}
}

func TestIsValidSampleRate(t *testing.T) {
	cases := []struct {
		hz   uint32
		want bool
	}{
		{hz: 225_000, want: false},   // C source rejects samp_rate <= 225_000
		{hz: 225_001, want: true},    // first allowed value just above floor
		{hz: 250_000, want: true},    // common sub-rate
		{hz: 300_000, want: true},    // boundary: 300_000 is OK (gap starts at >300_000)
		{hz: 300_001, want: false},   // inside forbidden gap
		{hz: 600_000, want: false},   // middle of gap
		{hz: 900_000, want: false},   // upper edge of gap (still forbidden — <= 900_000)
		{hz: 900_001, want: true},    // just above gap
		{hz: 2_400_000, want: true},  // default rate
		{hz: 3_200_000, want: true},  // upper limit
		{hz: 3_200_001, want: false}, // above ceiling
	}
	for _, tc := range cases {
		got := IsValidSampleRate(tc.hz)
		if got != tc.want {
			t.Errorf("IsValidSampleRate(%d) = %v, want %v", tc.hz, got, tc.want)
		}
	}
}

func TestSetSampleRate_RejectsOutOfRange(t *testing.T) {
	d := New(usb.NewMockTransport())
	for _, hz := range []uint32{0, 100_000, 225_000, 300_001, 900_000, 5_000_000} {
		_, err := d.SetSampleRate(hz)
		if err == nil {
			t.Errorf("SetSampleRate(%d) = nil, want error", hz)
		}
	}
}

func TestSetSampleRate_WritesDivisorAndResets(t *testing.T) {
	// Full transcript for 2.4 MS/s:
	// - demod write page=1 addr=0x9F val=0x0300 n=2  (divisor high half)
	// - demod write page=1 addr=0xA1 val=0x0000 n=2  (divisor low half)
	// - sample-freq correction at ppm=0 (two demod writes at 0x3F, 0x3E)
	// - soft-reset demod (writes 0x14 then 0x10 to page=1 addr=0x01)
	m := usb.NewMockTransport()
	var script []usb.CtrlExchange
	script = append(script, expectDemodWrite(1, 0x9F, 0x0300, 2)...)
	script = append(script, expectDemodWrite(1, 0xA1, 0x0000, 2)...)
	script = append(script, expectDemodWrite(1, 0x3F, 0x00, 1)...)
	script = append(script, expectDemodWrite(1, 0x3E, 0x00, 1)...)
	script = append(script, expectDemodWrite(1, 0x01, 0x14, 1)...)
	script = append(script, expectDemodWrite(1, 0x01, 0x10, 1)...)
	m.Script = script

	d := New(m)
	got, err := d.SetSampleRate(2_400_000)
	if err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if got != 2_400_000 {
		t.Errorf("actual rate = %d, want 2_400_000", got)
	}
	if m.Err != nil {
		t.Errorf("mock err = %v", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining script entries = %d, want 0", m.Remaining())
	}
	if d.GetSampleRate() != 2_400_000 {
		t.Errorf("GetSampleRate = %d, want 2_400_000", d.GetSampleRate())
	}
}

func TestSetIFFreq_MathMatchesLibrtlsdr(t *testing.T) {
	// For freq=0 Hz the IF-freq divisor is 0; all three demod registers
	// receive 0x00. The 22-bit value is the negative of the freq.
	m := usb.NewMockTransport()
	script := []usb.CtrlExchange{}
	script = append(script, expectDemodWrite(1, 0x19, 0x00, 1)...)
	script = append(script, expectDemodWrite(1, 0x1A, 0x00, 1)...)
	script = append(script, expectDemodWrite(1, 0x1B, 0x00, 1)...)
	m.Script = script

	d := New(m)
	if err := d.SetIFFreq(0); err != nil {
		t.Fatalf("SetIFFreq(0): %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestSetSampleFreqCorrection_Zero(t *testing.T) {
	// ppm=0 → offs=0 → both writes land 0x00.
	m := usb.NewMockTransport()
	script := []usb.CtrlExchange{}
	script = append(script, expectDemodWrite(1, 0x3F, 0x00, 1)...)
	script = append(script, expectDemodWrite(1, 0x3E, 0x00, 1)...)
	m.Script = script

	d := New(m)
	if err := d.SetSampleFreqCorrection(0); err != nil {
		t.Fatalf("SetSampleFreqCorrection(0): %v", err)
	}
}

func TestSetSampleFreqCorrection_Nonzero(t *testing.T) {
	// ppm=50 → offs = -50 * 2^24 / 1_000_000 = -838
	// Truncating to int16: -838 = 0xFCBA (two's complement).
	// Low byte 0xBA → write to 0x3F.
	// High byte sign-extended: 0xFC, masked to 6 bits: 0x3C → write to 0x3E.
	m := usb.NewMockTransport()
	script := []usb.CtrlExchange{}
	script = append(script, expectDemodWrite(1, 0x3F, 0xBA, 1)...)
	script = append(script, expectDemodWrite(1, 0x3E, 0x3C, 1)...)
	m.Script = script

	d := New(m)
	if err := d.SetSampleFreqCorrection(50); err != nil {
		t.Fatalf("SetSampleFreqCorrection(50): %v", err)
	}
	if m.Err != nil {
		t.Errorf("mock err: %v", m.Err)
	}
}

func TestResetBuffer_SequenceMatchesLibrtlsdr(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		expectBlockWrite(BlockUSB, USBEpaCtl, 0x1002, 2),
		expectBlockWrite(BlockUSB, USBEpaCtl, 0x0000, 2),
	}
	d := New(m)
	if err := d.ResetBuffer(); err != nil {
		t.Fatalf("ResetBuffer: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestSetFIRDefault_TwentyDemodWrites(t *testing.T) {
	// Pre-compute the expected byte sequence the FIR packing math
	// produces against firDefault.
	expected := buildFIRWireBytes(t, firDefault)
	m := usb.NewMockTransport()
	for i, b := range expected {
		m.Script = append(m.Script, expectDemodWrite(1, 0x1C+uint16(i), uint16(b), 1)...)
	}

	d := New(m)
	if err := d.SetFIRDefault(); err != nil {
		t.Fatalf("SetFIRDefault: %v", err)
	}
	if m.Err != nil {
		t.Errorf("mock err: %v", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestSetFIR_OutOfRange8BitTap(t *testing.T) {
	var bad [20]int16 = firDefault
	bad[2] = 200 // outside 8-bit signed range
	if err := New(usb.NewMockTransport()).SetFIR(bad); err == nil {
		t.Fatal("expected error for out-of-range 8-bit tap")
	}
}

func TestSetFIR_OutOfRange12BitTap(t *testing.T) {
	var bad [20]int16 = firDefault
	bad[10] = 3000 // outside 12-bit signed range (±2048)
	if err := New(usb.NewMockTransport()).SetFIR(bad); err == nil {
		t.Fatal("expected error for out-of-range 12-bit tap")
	}
}

// buildFIRWireBytes recomputes the FIR packing in the test (independent of
// production code) so a packing-math bug shows up as a script mismatch
// rather than silently passing through identical logic.
func buildFIRWireBytes(t *testing.T, coeffs [20]int16) []byte {
	t.Helper()
	var out [20]byte
	for i := 0; i < 8; i++ {
		out[i] = byte(coeffs[i])
	}
	for i := 0; i < 4; i++ {
		v0 := coeffs[8+i*2]
		v1 := coeffs[8+i*2+1]
		out[8+i*3] = byte(v0 >> 4)
		out[8+i*3+1] = byte((v0 << 4) | ((v1 >> 8) & 0x0F))
		out[8+i*3+2] = byte(v1)
	}
	return out[:]
}

func TestInitBaseband_FullSequence(t *testing.T) {
	// Builds the entire init transcript and asserts InitBaseband walks
	// it in order. If anyone reorders rtlsdr_init_baseband upstream
	// without porting the change, this test fails noisily.
	m := usb.NewMockTransport()
	var script []usb.CtrlExchange

	// Steps 0..14 (before FIR upload).
	for i := 0; i < 15; i++ {
		s := initBasebandSteps[i]
		if s.demod {
			script = append(script, expectDemodWrite(s.page, s.addr, s.val, s.n)...)
		} else {
			script = append(script, expectBlockWrite(s.block, s.addr, s.val, s.n))
		}
	}
	// FIR upload — same expected wire bytes as TestSetFIRDefault.
	firBytes := buildFIRWireBytes(t, firDefault)
	for i, b := range firBytes {
		script = append(script, expectDemodWrite(1, 0x1C+uint16(i), uint16(b), 1)...)
	}
	// Steps 15..end (after FIR upload).
	for i := 15; i < len(initBasebandSteps); i++ {
		s := initBasebandSteps[i]
		script = append(script, expectDemodWrite(s.page, s.addr, s.val, s.n)...)
	}
	m.Script = script

	d := New(m)
	if err := d.InitBaseband(); err != nil {
		t.Fatalf("InitBaseband: %v", err)
	}
	if m.Err != nil {
		t.Errorf("mock err: %v", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestWarmupUSBSysctl_SingleWrite(t *testing.T) {
	// Mirrors librtlsdr's rtlsdr_open dummy-write probe. Wire bytes
	// must be identical to initBasebandSteps[0] (BlockUSB / USBSysctl
	// / 0x09 / n=1) so the chip happily accepts both this probe and
	// the InitBaseband write that follows.
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		expectBlockWrite(BlockUSB, USBSysctl, 0x09, 1),
	}
	d := New(m)
	if err := d.WarmupUSBSysctl(); err != nil {
		t.Fatalf("WarmupUSBSysctl: %v", err)
	}
	if m.Err != nil {
		t.Errorf("mock err: %v", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestDeinitBaseband_ClearsDemodCtl(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		expectBlockWrite(BlockSys, SysDemodCtl, 0x20, 1),
	}
	d := New(m)
	if err := d.DeinitBaseband(); err != nil {
		t.Fatalf("DeinitBaseband: %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}
