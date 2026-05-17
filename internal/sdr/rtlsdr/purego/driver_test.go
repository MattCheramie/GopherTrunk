package purego

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

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
// Termination is forced via an unrelated ErrTimeout on the first
// InitBaseband write so the script stays minimal.
func TestOpenDevice_WarmupSucceedsFirstTry(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(nil),
		// Make InitBaseband's first write (also USBSysctl=0x09 — the
		// same wire bytes as the warmup) fail with a non-EPIPE error
		// so openDevice unwinds cleanly without forcing us to script
		// the full init flood + tuner detect.
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2000,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x09},
			Err:      usb.ErrTimeout,
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

// Regression for issue #248: when the warmup write returns EPIPE, the
// driver must call transport.Reset, re-claim interface 0, and retry the
// warmup once. The retry succeeds and openDevice proceeds — we again
// terminate early via ErrTimeout on the first InitBaseband write.
func TestOpenDevice_WarmupEPIPETriggersResetAndRetry(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(syscall.EPIPE),
		warmupUSBSysctlExchange(nil),
		{
			In:       false,
			BRequest: 0,
			WValue:   0x2000,
			WIndex:   uint16(1)<<8 | 0x10,
			Data:     []byte{0x09},
			Err:      usb.ErrTimeout,
		},
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-retry"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected init-baseband timeout to terminate the test")
	}
	if !strings.Contains(err.Error(), "init baseband") {
		t.Errorf("err = %v, want substring \"init baseband\" (proves warmup retry succeeded and InitBaseband ran)", err)
	}
	if m.ResetCalls != 1 {
		t.Errorf("ResetCalls = %d, want 1 (EPIPE on warmup must trigger one USBDEVFS_RESET)", m.ResetCalls)
	}
	if m.ClaimCalls != 2 {
		t.Errorf("ClaimCalls = %d, want 2 (initial claim + post-reset re-claim)", m.ClaimCalls)
	}
}

// Regression for issue #248: when both warmup attempts return EPIPE,
// openDevice must surface the wrapped error with the tunerBringupHint
// appended — that's the actionable message the user sees and which
// points them at the DVB / power / cable workarounds.
func TestOpenDevice_WarmupEPIPETwiceReturnsHintError(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		warmupUSBSysctlExchange(syscall.EPIPE),
		warmupUSBSysctlExchange(syscall.EPIPE),
	}
	desc := usb.Descriptor{VID: 0x0bda, PID: 0x2838, Serial: "test-warmup-fail"}
	_, err := openDevice(m, desc, 0)
	if err == nil {
		t.Fatal("openDevice succeeded; expected EPIPE-twice to fail open")
	}
	if !errors.Is(err, syscall.EPIPE) {
		t.Errorf("err = %v, want errors.Is(err, syscall.EPIPE) (the underlying cause must remain inspectable)", err)
	}
	if !strings.Contains(err.Error(), "USB warmup") {
		t.Errorf("err = %v, want substring \"USB warmup\" (identifies the failing stage)", err)
	}
	if !strings.Contains(err.Error(), "dvb_usb_rtl28xxu") {
		t.Errorf("err = %v, want substring \"dvb_usb_rtl28xxu\" (proves tunerBringupHint was appended)", err)
	}
	if m.ResetCalls != 1 {
		t.Errorf("ResetCalls = %d, want 1 (one reset attempt between the two warmup tries)", m.ResetCalls)
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
			name:  "ETIMEDOUT_not_hinted",
			err:   fmt.Errorf("wrap: %w", usb.ErrTimeout),
			empty: true,
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
