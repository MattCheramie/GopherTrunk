package airspy

import (
	"os"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

const (
	reqBoardIDReadCoreProbe       uint8 = 9
	reqVersionStringReadCoreProbe uint8 = 10
)

// TestRealHardware_USBCoreVendorReadProbe checks whether basic read-only
// Airspy vendor requests succeed on the current host/device path.
func TestRealHardware_USBCoreVendorReadProbe(t *testing.T) {
	requireRealAirspy(t)
	if !envBool(airspyRealDiagEnv) {
		t.Skipf("set %s=1 to run core vendor read probe", airspyRealDiagEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyRealSerialEnv))
	enum := usb.DefaultEnumerator()
	descs, err := enum.List(vidAirspy, pidAirspy)
	if err != nil {
		t.Fatalf("usb enumerate: %v", err)
	}
	if len(descs) == 0 {
		t.Fatalf("usb enumerate returned no Airspy descriptors")
	}

	desc := descs[0]
	if serialHint != "" {
		matched := false
		for _, d := range descs {
			if d.Serial == serialHint || strings.Contains(d.Serial, serialHint) {
				desc = d
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("no Airspy descriptor matched %q; found serials: %v", serialHint, collectUSBSerials(descs))
		}
	}

	t.Logf("core probe backend=%s path=%q serial=%q", enum.Name(), desc.Path, desc.Serial)
	tr, err := enum.Open(desc)
	if err != nil {
		t.Fatalf("usb open: %v", err)
	}
	defer tr.Close()
	if err := tr.ClaimInterface(0); err != nil {
		t.Fatalf("usb claim interface 0: %v", err)
	}
	defer tr.ReleaseInterface(0)

	cases := []struct {
		name           string
		bRequest       uint8
		wValue, wIndex uint16
		n              int
	}{
		{name: "board-v0-i0", bRequest: reqBoardIDReadCoreProbe, wValue: 0, wIndex: 0, n: 1},
		{name: "board-v0-i1", bRequest: reqBoardIDReadCoreProbe, wValue: 0, wIndex: 1, n: 1},
		{name: "ver-v0-i0", bRequest: reqVersionStringReadCoreProbe, wValue: 0, wIndex: 0, n: 64},
		{name: "ver-v0-i1", bRequest: reqVersionStringReadCoreProbe, wValue: 0, wIndex: 1, n: 64},
	}

	okCount := 0
	var lastErr error
	for _, tc := range cases {
		buf, inErr := tr.ControlIn(tc.bRequest, tc.wValue, tc.wIndex, tc.n, controlTimeoutMs)
		if inErr == nil {
			okCount++
			t.Logf("core IN %s ok: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d gotLen=%d", tc.name, tc.bRequest, tc.wValue, tc.wIndex, tc.n, len(buf))
		} else {
			lastErr = inErr
			t.Logf("core IN %s failed: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d err=%v", tc.name, tc.bRequest, tc.wValue, tc.wIndex, tc.n, inErr)
		}
	}

	if okCount == 0 {
		t.Fatalf("core vendor probe failed for all variants (last err=%v)", lastErr)
	}
}
