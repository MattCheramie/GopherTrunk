package purego

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestDriverNameIsRtlsdr(t *testing.T) {
	d := New(nil)
	if got, want := d.Name(), "rtlsdr"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if DriverName != "rtlsdr" {
		t.Errorf("DriverName const = %q, want rtlsdr", DriverName)
	}
}

func TestLookupKnown_HitsRealtekDefaults(t *testing.T) {
	// 0x0bda:0x2832 and 0x0bda:0x2838 are the dominant generic
	// RTL2832U IDs — must always match.
	for _, c := range []struct {
		vid, pid uint16
		wantSub  string
	}{
		{0x0bda, 0x2832, "Generic RTL2832U"},
		{0x0bda, 0x2838, "Generic RTL2832U OEM"},
	} {
		k := lookupKnown(c.vid, c.pid)
		if k == nil {
			t.Errorf("lookupKnown(0x%04x, 0x%04x) = nil, want match", c.vid, c.pid)
			continue
		}
		if !strings.Contains(k.Name, c.wantSub) {
			t.Errorf("lookupKnown(0x%04x, 0x%04x).Name = %q, want substring %q", c.vid, c.pid, k.Name, c.wantSub)
		}
	}
}

func TestLookupKnown_MissReturnsNil(t *testing.T) {
	if k := lookupKnown(0x1d50, 0x6089); k != nil { // HackRF VID/PID
		t.Errorf("lookupKnown(HackRF) = %+v, want nil (not an RTL-SDR)", k)
	}
}

func TestEnumerate_FiltersToKnownDevices(t *testing.T) {
	mock := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			// Two RTL-SDRs (one populated product string, one bare).
			{Bus: 1, Address: 4, VID: 0x0bda, PID: 0x2838, Serial: "00000001", Manufacturer: "Realtek", Product: "RTL2838UHIDIR"},
			{Bus: 1, Address: 5, VID: 0x0bda, PID: 0x2832, Serial: "00000002", Manufacturer: "Realtek"},
			// Not an RTL-SDR — must be skipped.
			{Bus: 1, Address: 6, VID: 0x1d50, PID: 0x6089, Serial: "hackrf"},
		},
	}
	d := New(mock)
	got, err := d.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Enumerate returned %d entries, want 2 (HackRF must be filtered)", len(got))
	}
	if got[0].Driver != "rtlsdr" {
		t.Errorf("Info.Driver = %q, want rtlsdr", got[0].Driver)
	}
	if got[0].Index != 0 || got[1].Index != 1 {
		t.Errorf("indices = (%d, %d), want (0, 1)", got[0].Index, got[1].Index)
	}
	// First device populated iProduct → use it verbatim.
	if got[0].Product != "RTL2838UHIDIR" {
		t.Errorf("Info.Product[0] = %q, want RTL2838UHIDIR (device-reported)", got[0].Product)
	}
	// Second device has empty iProduct → fall back to the friendly name.
	if !strings.Contains(got[1].Product, "RTL2832U") {
		t.Errorf("Info.Product[1] = %q, want fallback friendly name", got[1].Product)
	}
}

func TestEnumerate_PopulatesDetectCacheForOpen(t *testing.T) {
	mock := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{VID: 0x0bda, PID: 0x2838, Serial: "00000001"},
		},
	}
	d := New(mock)
	if _, err := d.Enumerate(); err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if got := len(d.detectCache); got != 1 {
		t.Errorf("detectCache size = %d, want 1", got)
	}
	if d.detectCache[0].Serial != "00000001" {
		t.Errorf("cached serial = %q, want 00000001", d.detectCache[0].Serial)
	}
}

func TestOpen_BadIndexReturnsError(t *testing.T) {
	d := New(&usb.MockEnumerator{})
	if _, err := d.Open(0); err == nil {
		t.Error("Open(0) on empty cache returned nil, want error")
	}
	if _, err := d.Open(-1); err == nil {
		t.Error("Open(-1) returned nil, want error")
	}
}

func TestEnumerator_DefaultFallback(t *testing.T) {
	// New(nil) should return a Driver that uses DefaultEnumerator.
	d := New(nil)
	enum := d.enumerator()
	if enum == nil {
		t.Fatal("default enumerator is nil")
	}
	// The platform's default enumerator name is OS-dependent
	// ("usbdevfs" on Linux, "winusb" on Windows, "macos-stub" on
	// macOS); we just assert it's non-empty.
	if enum.Name() == "" {
		t.Error("default enumerator Name() is empty")
	}
}

func TestKnownDevicesNoDuplicates(t *testing.T) {
	seen := map[uint32]string{}
	for _, k := range knownDevices {
		key := uint32(k.VID)<<16 | uint32(k.PID)
		if existing, ok := seen[key]; ok {
			t.Errorf("duplicate VID/PID 0x%04x:0x%04x: %q vs %q", k.VID, k.PID, existing, k.Name)
		}
		seen[key] = k.Name
	}
}

// TestKnownDevicesBiasTeeGPIODefault asserts the zero-value fallback
// the per-(VID, PID) bias-tee table relies on. The dominant RTL-SDR
// designs wire bias-tee to GPIO 0, so every entry in the table that
// doesn't override it must surface as GPIO 0 to keep the historical
// behaviour. New non-zero entries are deliberate overrides and not
// covered here.
func TestKnownDevicesBiasTeeGPIODefault(t *testing.T) {
	for _, k := range knownDevices {
		if k.BiasTeeGPIO > 7 {
			t.Errorf("%s: BiasTeeGPIO = %d, must be 0..7", k.Name, k.BiasTeeGPIO)
		}
	}
	// The standard Realtek reference VID/PID — operators rely on the
	// historical GPIO 0 mapping.
	k := lookupKnown(0x0bda, 0x2832)
	if k == nil {
		t.Fatalf("generic 0x0bda:0x2832 missing from knownDevices")
	}
	if k.BiasTeeGPIO != 0 {
		t.Errorf("generic RTL2832U bias-tee GPIO = %d, want 0 (regression)", k.BiasTeeGPIO)
	}
}

// warmupUSBSysctlExchange returns the single mock CtrlExchange that
// matches the wire bytes WarmupUSBSysctl emits — a vendor-OUT write to
// BlockUSB / USBSysctl with payload 0x09. Err overrides the mock's
// response so tests can simulate EPIPE recoveries.
func warmupUSBSysctlExchange(simulateErr error) usb.CtrlExchange {
	return usb.CtrlExchange{
		In:       false,
		BRequest: 0,
		WValue:   0x2000,              // USBSysctl
		WIndex:   uint16(1)<<8 | 0x10, // BlockUSB<<8 | 0x10
		Data:     []byte{0x09},
		Err:      simulateErr,
	}
}

// Regression for issue #248: the warmup probe in openDevice must run
// exactly once on a healthy device and must not trigger a USB reset.
// Termination is forced via ErrClosed on the first InitBaseband write
// so the script stays minimal — ErrClosed is explicitly non-resetable
// (see isBringupResetable) and unwinds the bring-up immediately.
func TestOpenDevice_WarmupSucceedsFirstTry(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),
		// Make InitBaseband's first write (also USBSysctl=0x09 — the
		// same wire bytes as the warmup) fail with a non-resetable
		// error so openDevice unwinds cleanly without forcing us to
		// script the full init flood + tuner detect.
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2000,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x09},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-ok"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband timeout to terminate the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves warmup passed and InitBaseband ran)", err)
	}
	if m.ResetCalls != 0 {
		t.Errorf("ResetCalls = %d, want 0 (healthy warmup must not reset)", m.ResetCalls)
	}
	if m.ClaimCalls != 1 {
		t.Errorf("ClaimCalls = %d, want 1 (single ClaimInterface on healthy path)", m.ClaimCalls)
	}
}

// Regression for issue #395: WarmupUSBSysctl and the first InitBaseband
// write are byte-identical (BlockUSB / USBSysctl / 0x09). On Windows
// through the direct WinUsb_ControlTransfer syscall this back-to-back
// pair races some clone-dongle firmware and trips ERROR_GEN_FAILURE on
// the second write. runBringup must sleep warmupSettleDuration between
// the two transfers so the dongle has time to settle. The test asserts
// the wall-clock gap from openDevice entry to the InitBaseband
// terminator error is at least warmupSettleDuration, then restores
// the original value (the test temporarily widens it to 75 ms so the
// timing observation stays robust above scheduler noise).
func TestOpenDevice_WarmupToInitBasebandSettle_Issue395(t *testing.T) {
	original := warmupSettleDuration
	warmupSettleDuration = 75 * time.Millisecond
	t.Cleanup(func() { warmupSettleDuration = original })

	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),
		// Fail the first InitBaseband write with ErrClosed (non-resetable)
		// so openDevice unwinds immediately on attempt 0 — we just want
		// to measure that the settle ran between the warmup and step 0.
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2000,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x09},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-settle"}
	start := time.Now()
	_, err := openDevice(m, desc, 0)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves warmup passed and InitBaseband ran)", err)
	}
	if elapsed < warmupSettleDuration {
		t.Errorf("openDevice took %v, want >= warmupSettleDuration (%v) — the inter-transfer settle did not run", elapsed, warmupSettleDuration)
	}
	if m.ResetCalls != 0 {
		t.Errorf("ResetCalls = %d, want 0 (the settle path must not invoke reset; the failure here is non-resetable ErrClosed)", m.ResetCalls)
	}
}

// Regression for the Windows ERROR_GEN_FAILURE warmup failure
// (follow-up to issue #395): the warmup write is librtlsdr's sacrificial
// dummy write. When it NAKs (ErrPipeStalled, the WinUSB mapping of
// ERROR_GEN_FAILURE), runBringup must SWALLOW the error and proceed
// straight to InitBaseband — whose byte-identical step 0 is the transfer
// that actually has to land. Resetting + re-issuing the warmup (the old
// behaviour) only re-armed the same first-transfer NAK and the dongle
// never came up. This pins that a warmup NAK followed by a healthy
// InitBaseband step 0 reaches step 1 with NO reset.
func TestOpenDevice_WarmupNonFatal_ProceedsToInitBaseband(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		// Sacrificial warmup NAKs with ERROR_GEN_FAILURE.
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
		// InitBaseband step 0 (byte-identical to the warmup) now lands —
		// the cold first-transfer NAK was absorbed by the warmup.
		warmupUSBSysctlExchange(nil),
		// InitBaseband step 1 (USBEpaMaxpkt=0x0002, n=2). Fail it with a
		// non-resetable error to unwind the bring-up without scripting the
		// full init flood — proving we advanced past step 0.
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2158, // USBEpaMaxpkt
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x00, 0x02},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-nonfatal"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected the InitBaseband step-1 terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves the swallowed warmup let InitBaseband run)", err)
	}
	if !errors.Is(err, usb.ErrClosed) {
		t.Errorf("err = %v, want errors.Is(err, usb.ErrClosed) (the step-1 terminator)", err)
	}
	if m.ResetCalls != 0 {
		t.Errorf("ResetCalls = %d, want 0 (a swallowed warmup NAK must NOT trip the reset envelope)", m.ResetCalls)
	}
	if m.ClaimCalls != 1 {
		t.Errorf("ClaimCalls = %d, want 1 (no reset ⇒ single ClaimInterface)", m.ClaimCalls)
	}
}

// The warmup write is swallowed regardless of the error class it fails
// with — EPIPE (Linux), ErrTimeout (WinUSB cold-boot), ErrPipeStalled
// (WinUSB ERROR_GEN_FAILURE), or even ErrClosed. In every case bring-up
// advances to InitBaseband: no "USB warmup" stage error and no reset
// keyed off the warmup. Here InitBaseband step 0 is failed with a
// non-resetable ErrClosed so the bring-up unwinds immediately.
func TestOpenDevice_WarmupErrorSwallowedRegardlessOfClass(t *testing.T) {
	for _, warmupErr := range []error{syscall.EPIPE, usb.ErrTimeout, usb.ErrPipeStalled, usb.ErrClosed} {
		m := usb.NewMockTransport()
		m.Script = []usb.CtrlExchange{
			warmupUSBSysctlExchange(warmupErr),     // warmup NAKs (swallowed)
			warmupUSBSysctlExchange(usb.ErrClosed), // InitBaseband step 0: non-resetable terminator
		}
		desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-swallow"}
		_, err := openDevice(m, desc, 0)
		if err == nil {
			t.Fatalf("warmupErr=%v: openDevice succeeded; expected the InitBaseband terminator to fail", warmupErr)
		}
		if !strings.Contains(err.Error(), "init baseband") {
			t.Errorf("warmupErr=%v: err = %v, want substring \"init baseband\" (warmup must be swallowed, not surfaced)", warmupErr, err)
		}
		if strings.Contains(err.Error(), "USB warmup") {
			t.Errorf("warmupErr=%v: err = %v, must NOT contain \"USB warmup\" (the stage is no longer a hard gate)", warmupErr, err)
		}
		if m.ResetCalls != 0 {
			t.Errorf("warmupErr=%v: ResetCalls = %d, want 0 (warmup failure must not reset)", warmupErr, m.ResetCalls)
		}
		if m.ClaimCalls != 1 {
			t.Errorf("warmupErr=%v: ClaimCalls = %d, want 1", warmupErr, m.ClaimCalls)
		}
	}
}

// Regression for issue #248: EPIPE that surfaces AFTER warmup (so
// post-warmup, anywhere in InitBaseband / Detect / PrepareDemod /
// tuner.Init / SetIFFreq) must trigger the outer envelope's one-shot
// reset+retry of the entire bring-up. This test injects EPIPE on
// InitBaseband's first write, which has byte-identical wire bytes to
// the warmup probe but is a distinct call from openDevice's
// perspective — the outer envelope is the only path that retries
// past warmup. The retry succeeds the warmup but is terminated early
// via ErrTimeout on InitBaseband's first write (the second pass) so
// the script stays minimal.
func TestOpenDevice_BringupEPIPE_TriggersFullReset(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),           // pass 1: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 1: InitBaseband first write EPIPE
		warmupUSBSysctlExchange(nil),           // pass 2: warmup OK (post-reset)
		warmupUSBSysctlExchange(usb.ErrClosed), // pass 2: non-resetable terminator
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-retry"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves the retry reached InitBaseband)", err)
	}
	if m.ResetCalls != 1 {
		t.Errorf("ResetCalls = %d, want 1 (one reset between the two bring-up attempts)", m.ResetCalls)
	}
	if m.ClaimCalls != 2 {
		t.Errorf("ClaimCalls = %d, want 2 (initial claim + post-reset re-claim)", m.ClaimCalls)
	}
}

// Regression for issue #248: when EPIPE recurs on every retry pass of
// the bring-up, openDevice must surface the wrapped error with the
// tunerBringupHint appended. Up to four USBDEVFS_RESETs are allowed
// per Open call — the envelope is bounded, never an unbounded loop.
func TestOpenDevice_BringupEPIPEFiveTimes_ReturnsHintError(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),           // pass 1: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 1: InitBaseband EPIPE
		warmupUSBSysctlExchange(nil),           // pass 2: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 2: InitBaseband EPIPE
		warmupUSBSysctlExchange(nil),           // pass 3: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 3: InitBaseband EPIPE
		warmupUSBSysctlExchange(nil),           // pass 4: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 4: InitBaseband EPIPE
		warmupUSBSysctlExchange(nil),           // pass 5: warmup OK
		warmupUSBSysctlExchange(syscall.EPIPE), // pass 5: InitBaseband EPIPE again
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-fail"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected EPIPE-five-times to fail open")
	}
	if !errors.Is(err, syscall.EPIPE) {
		t.Errorf("err = %v, want errors.Is(err, syscall.EPIPE)", err)
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (identifies the failing stage)", err)
	}
	if !strings.Contains(err.Error(), "dvb_usb_rtl28xxu") {
		t.Errorf("err = %v, want substring \"dvb_usb_rtl28xxu\" (proves tunerBringupHint was appended)", err)
	}
	if m.ResetCalls != 4 {
		t.Errorf("ResetCalls = %d, want 4 (bounded envelope: four resets, five attempts)", m.ResetCalls)
	}
	if m.ClaimCalls != 5 {
		t.Errorf("ClaimCalls = %d, want 5 (initial claim + four post-reset re-claims)", m.ClaimCalls)
	}
}

// Regression for issue #248: tuner-init failures that look like the
// I2C-bridge-can't-reach-tuner case (EPIPE from the first burst, or
// the device disappearing mid-bringup) must carry remediation guidance
// pointing at the DVB kernel driver / USB power / cable workarounds.
// Unrelated errors must pass through untouched.
func TestTunerBringupHint(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantSub []string // substrings that must appear in the hint
		empty   bool     // true ⇒ hint must be exactly ""
	}{
		{
			name:    "EPIPE_through_wrap",
			err:     fmt.Errorf("rtl2832u: I2CWrite addr=0x34: %w", syscall.EPIPE),
			wantSub: []string{"dvb_usb_rtl28xxu", "install-linux.html#troubleshooting"},
		},
		{
			name:    "ErrDeviceGone",
			err:     fmt.Errorf("transport: %w", usb.ErrDeviceGone),
			wantSub: []string{"dvb_usb_rtl28xxu", "install-linux.html#troubleshooting"},
		},
		{
			name:  "nil_error",
			err:   nil,
			empty: true,
		},
		{
			name:  "unrelated_error",
			err:   errors.New("some other failure"),
			empty: true,
		},
		{
			name:    "ErrTimeout_through_wrap",
			err:     fmt.Errorf("wrap: %w", usb.ErrTimeout),
			wantSub: []string{"Zadig", "install-windows.html#troubleshooting"},
		},
		{
			name:    "ErrPipeStalled_through_wrap",
			err:     fmt.Errorf("winusb: device rejected request: %w", usb.ErrPipeStalled),
			wantSub: []string{"control pipe stalled", "ERROR_GEN_FAILURE", "issue #395", "sdr doctor", "install-windows.html#troubleshooting"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tunerBringupHint(c.err)
			if c.empty {
				if got != "" {
					t.Fatalf("tunerBringupHint(%v) = %q, want empty", c.err, got)
				}
				return
			}
			for _, sub := range c.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("tunerBringupHint(%v) = %q, missing substring %q", c.err, got, sub)
				}
			}
		})
	}
}

// Mode B (Windows cold-boot, the genuinely-wedged case): the warmup
// NAKs AND InitBaseband's byte-identical step 0 NAKs too (ErrPipeStalled
// from ERROR_GEN_FAILURE). The warmup failure is swallowed, but the
// step-0 failure is caught by the outer envelope, which resets
// (WinUsb_ResetPipe / re-open) and retries the whole bring-up; the
// post-reset pass lands step 0 and proceeds. This is the recovery path
// that survives moving the hard gate from warmup to InitBaseband — it
// also covers the cold-boot ErrTimeout class, which is resetable the
// same way.
func TestOpenDevice_WarmupAndInitBasebandStall_ResetRecovers(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 1: warmup NAK (swallowed)
		warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 1: InitBaseband step 0 NAK → resetable
		warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 2: warmup NAK (swallowed, post-reset)
		warmupUSBSysctlExchange(nil),                // pass 2: InitBaseband step 0 OK
		// pass 2: InitBaseband step 1 (USBEpaMaxpkt) non-resetable terminator.
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2158,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x00, 0x02},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-and-init-stall"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected the InitBaseband step-1 terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\"", err)
	}
	if m.ResetCalls != 1 {
		t.Errorf("ResetCalls = %d, want 1 (the step-0 stall on the cold pass triggers one reset; warmup NAKs do not)", m.ResetCalls)
	}
	if m.ClaimCalls != 2 {
		t.Errorf("ClaimCalls = %d, want 2 (initial claim + post-reset re-claim)", m.ClaimCalls)
	}
}

// The cold-boot symptom that prompted the fix — warmup succeeds, then
// the byte-identical first InitBaseband write stalls with
// ERROR_GEN_FAILURE / ErrPipeStalled. The bring-up envelope must reset
// and retry the whole flow.
func TestOpenDevice_BringupPipeStalled_TriggersFullReset(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),                // pass 1: warmup OK
		warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 1: InitBaseband step 0 stalls
		warmupUSBSysctlExchange(nil),                // pass 2: warmup OK (post clear-halt)
		warmupUSBSysctlExchange(usb.ErrClosed),      // pass 2: non-resetable terminator
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-pipestalled-retry"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves the retry reached InitBaseband)", err)
	}
	if m.ResetCalls != 1 {
		t.Errorf("ResetCalls = %d, want 1 (one clear-halt between the two bring-up attempts)", m.ResetCalls)
	}
	if m.ClaimCalls != 2 {
		t.Errorf("ClaimCalls = %d, want 2 (initial claim + post-reset re-claim)", m.ClaimCalls)
	}
}

// When ErrPipeStalled recurs on every retry pass, the surfaced error
// must carry the clone-dongle / Zadig / unplug-and-replug hint pointing
// at issue #395.
func TestOpenDevice_BringupPipeStalledFiveTimes_ReturnsHintError(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
		warmupUSBSysctlExchange(nil),
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
		warmupUSBSysctlExchange(nil),
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
		warmupUSBSysctlExchange(nil),
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
		warmupUSBSysctlExchange(nil),
		warmupUSBSysctlExchange(usb.ErrPipeStalled),
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-pipestalled-fail"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected ErrPipeStalled-five-times to fail open")
	}
	if !errors.Is(err, usb.ErrPipeStalled) {
		t.Errorf("err = %v, want errors.Is(err, usb.ErrPipeStalled)", err)
	}
	if !strings.Contains(err.Error(), "control pipe stalled") {
		t.Errorf("err = %v, want substring \"control pipe stalled\" (proves the clone-dongle hint was appended)", err)
	}
	if !strings.Contains(err.Error(), "ERROR_GEN_FAILURE") {
		t.Errorf("err = %v, want substring \"ERROR_GEN_FAILURE\" (proves the Windows-specific hint was appended)", err)
	}
	if !strings.Contains(err.Error(), "issue #395") {
		t.Errorf("err = %v, want substring \"issue #395\" (proves the issue ref was appended)", err)
	}
	if !strings.Contains(err.Error(), "sdr doctor") {
		t.Errorf("err = %v, want substring \"sdr doctor\" (proves the WinUSB-binding remediation was appended)", err)
	}
	if m.ResetCalls != 4 {
		t.Errorf("ResetCalls = %d, want 4 (bounded envelope: four resets, five attempts)", m.ResetCalls)
	}
	if m.ClaimCalls != 5 {
		t.Errorf("ClaimCalls = %d, want 5 (initial claim + four post-reset re-claims)", m.ClaimCalls)
	}
}

// Regression: when ErrTimeout recurs on every bring-up pass (warmup
// swallowed, then InitBaseband step 0 times out), the surfaced error
// must carry the Windows-aware hint that points at the WinUSB / Zadig
// step. Bounded to four resets per Open call.
func TestOpenDevice_BringupTimeoutFiveTimesReturnsHintError(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrTimeout), // pass 1
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrTimeout), // pass 2
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrTimeout), // pass 3
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrTimeout), // pass 4
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrTimeout), // pass 5
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-timeout-fail"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected ErrTimeout-five-times to fail open")
	}
	if !errors.Is(err, usb.ErrTimeout) {
		t.Errorf("err = %v, want errors.Is(err, usb.ErrTimeout) (the underlying cause must remain inspectable)", err)
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (identifies the failing stage)", err)
	}
	if !strings.Contains(err.Error(), "Zadig") {
		t.Errorf("err = %v, want substring \"Zadig\" (proves the Windows-aware tunerBringupHint was appended)", err)
	}
	if m.ResetCalls != 4 {
		t.Errorf("ResetCalls = %d, want 4 (bounded envelope: four resets, five attempts)", m.ResetCalls)
	}
}

// Regression: a clone dongle on Windows can require two full resets to
// clear a wedged firmware state (the first WinUsb_Initialize rebind
// doesn't always settle the device, but the second one does). With the
// hard gate now on InitBaseband step 0, the envelope must keep retrying
// after the first reset and let a later pass succeed. Each pass: warmup
// swallowed, then InitBaseband step 0 stalls (passes 1-2) or lands
// (pass 3).
func TestOpenDevice_BringupStallRecoversOnSecondReset(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 1: step 0 stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 2: step 0 stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(nil), // pass 3: step 0 OK
		{ // pass 3: InitBaseband step 1 non-resetable terminator
			In:       false,
			BRequest: 0,
			WValue:   0x2158,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x00, 0x02},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-second-reset"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves the third attempt reached InitBaseband)", err)
	}
	if m.ResetCalls != 2 {
		t.Errorf("ResetCalls = %d, want 2 (one reset after each stalled InitBaseband step 0)", m.ResetCalls)
	}
	if m.ClaimCalls != 3 {
		t.Errorf("ClaimCalls = %d, want 3 (initial claim + two post-reset re-claims)", m.ClaimCalls)
	}
}

// Regression for issue #395: pins that the attempt-4 slot in the retry
// envelope is actually consulted. The bring-up must succeed on attempt 4
// (zero-indexed) after four InitBaseband step-0 stalls.
func TestOpenDevice_BringupStallRecoversOnFifthAttempt(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 1: stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 2: stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 3: stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(usb.ErrPipeStalled), // pass 4: stalls
		warmupUSBSysctlExchange(nil), warmupUSBSysctlExchange(nil), // pass 5: step 0 OK
		{ // pass 5: InitBaseband step 1 non-resetable terminator
			In:       false,
			BRequest: 0,
			WValue:   0x2158,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x00, 0x02},
			Err:      usb.ErrClosed,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-bringup-fifth-attempt"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband terminator to fail the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves the fifth attempt reached InitBaseband)", err)
	}
	if m.ResetCalls != 4 {
		t.Errorf("ResetCalls = %d, want 4 (one reset after each of the first four stalled step 0s)", m.ResetCalls)
	}
	if m.ClaimCalls != 5 {
		t.Errorf("ClaimCalls = %d, want 5 (initial claim + four post-reset re-claims)", m.ClaimCalls)
	}
}

// Regression for the "claim interface 0: device or resource busy"
// report: the USB transport detaches the bound DVB kernel driver and
// retries automatically, so an EBUSY that still reaches openDevice
// means another user-space process holds the dongle. claimBusyHint
// must turn that into actionable guidance; unrelated errors and nil
// must pass through with an empty hint.
func TestClaimBusyHint(t *testing.T) {
	busy := claimBusyHint(fmt.Errorf("rtlsdr: claim interface 0: %w", syscall.EBUSY))
	if !strings.Contains(busy, "another process") {
		t.Errorf("claimBusyHint(EBUSY) = %q, want it to mention another process", busy)
	}
	if !strings.Contains(busy, "install-linux.html#troubleshooting") {
		t.Errorf("claimBusyHint(EBUSY) = %q, want the troubleshooting link", busy)
	}
	if got := claimBusyHint(nil); got != "" {
		t.Errorf("claimBusyHint(nil) = %q, want empty", got)
	}
	if got := claimBusyHint(errors.New("unrelated")); got != "" {
		t.Errorf("claimBusyHint(unrelated) = %q, want empty", got)
	}
}

// Regression for the "claim interface 0: device or resource busy"
// report: when the USB transport cannot claim interface 0 because the
// dongle is genuinely held by another process, openDevice must surface
// the EBUSY wrapped with claimBusyHint — keeping the cause inspectable
// via errors.Is while pointing the operator at the fix.
func TestOpenDevice_ClaimBusySurfacesHint(t *testing.T) {
	m := usb.NewMockTransport()
	m.ClaimErr = syscall.EBUSY
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-claim-busy"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected claim EBUSY to fail open")
	}
	if !errors.Is(err, syscall.EBUSY) {
		t.Errorf("err = %v, want errors.Is(err, syscall.EBUSY)", err)
	}
	if !strings.Contains(err.Error(), "claim interface 0") {
		t.Errorf("err = %v, want substring \"claim interface 0\" (identifies the failing stage)", err)
	}
	if !strings.Contains(err.Error(), "another process") {
		t.Errorf("err = %v, want the claimBusyHint guidance appended", err)
	}
	if m.ResetCalls != 0 {
		t.Errorf("ResetCalls = %d, want 0 (a claim failure must not trigger a USB reset)", m.ResetCalls)
	}
}

// Regression: a non-resetable error class (here usb.ErrClosed) on a
// post-warmup transfer — InitBaseband step 0 — must surface immediately
// without any USB reset. Reset is the wrong hammer for "transport
// already closed" and would obscure the real cause. (Warmup-stage
// failures are now swallowed entirely; see
// TestOpenDevice_WarmupErrorSwallowedRegardlessOfClass.)
func TestOpenDevice_InitBasebandNonResetableErrorDoesNotReset(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),           // warmup OK
		warmupUSBSysctlExchange(usb.ErrClosed), // InitBaseband step 0: non-resetable
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-init-closed"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected ErrClosed to fail open")
	}
	if !errors.Is(err, usb.ErrClosed) {
		t.Errorf("err = %v, want errors.Is(err, usb.ErrClosed)", err)
	}
	if m.ResetCalls != 0 {
		t.Errorf("ResetCalls = %d, want 0 (non-resetable error must not trigger reset)", m.ResetCalls)
	}
}

// Regression for the Windows ERROR_GEN_FAILURE triage path: when
// bring-up ultimately fails, openDevice must append the transport's
// Diagnostics (bound driver, descriptors, control-IN read probe) to the
// error so one `gophertrunk sdr list --probe` run captures everything
// needed to diagnose a dongle that rejects control transfers even with
// WinUSB reportedly bound. The mock stands in for the WinUSB transport
// via the usb.Diagnoser interface.
func TestOpenDevice_AppendsTransportDiagnosticsOnFailure(t *testing.T) {
	m := usb.NewMockTransport()
	m.Diag = "bound driver: service=\"libusbK\" (winusb-bound=false)\n"
	// Persistent stall on every InitBaseband step 0 (warmup swallowed),
	// across all five attempts: 2 transfers per attempt × 5 attempts.
	m.Script = make([]usb.CtrlExchange, 0, 10)
	for i := 0; i < 5; i++ {
		m.Script = append(m.Script,
			warmupUSBSysctlExchange(usb.ErrPipeStalled), // warmup (swallowed)
			warmupUSBSysctlExchange(usb.ErrPipeStalled), // InitBaseband step 0 (resetable)
		)
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-diag"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected persistent stall to fail open")
	}
	if !strings.Contains(err.Error(), "USB diagnostics") {
		t.Errorf("err = %v, want the diagnostics block appended", err)
	}
	if !strings.Contains(err.Error(), "libusbK") {
		t.Errorf("err = %v, want the bound-driver diagnostic included", err)
	}
	// The underlying cause must still be inspectable through the wrap.
	if !errors.Is(err, usb.ErrPipeStalled) {
		t.Errorf("err = %v, want errors.Is(err, usb.ErrPipeStalled) after diagnostics wrap", err)
	}
}

// A transport that produces no diagnostics (the non-Windows case, Diag
// empty) must leave the error message unchanged — no stray diagnostics
// header.
func TestOpenDevice_NoDiagnosticsWhenTransportSilent(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),           // warmup OK
		warmupUSBSysctlExchange(usb.ErrClosed), // InitBaseband step 0: non-resetable
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-no-diag"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected ErrClosed to fail open")
	}
	if strings.Contains(err.Error(), "USB diagnostics") {
		t.Errorf("err = %v, must NOT contain the diagnostics header when the transport is silent", err)
	}
}
