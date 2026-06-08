package purego

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// newDeviceWithR828D builds a Device whose tuner is a real R82xx bound to
// an R828D. The tuner does no USB I/O for the methods under test
// (SetBlogV4 / TunerDiag only touch in-memory state), so the mock
// transport is never exercised.
func newDeviceWithR828D(t *testing.T) *Device {
	t.Helper()
	mock := usb.NewMockTransport()
	demod := rtl2832u.New(mock)
	r82 := tuners.NewR82xx(demod, 0x74, tuners.TypeR828D)
	return &Device{
		transport: mock,
		demod:     demod,
		tuner:     r82,
		info:      sdr.Info{Driver: DriverName, Index: 0, Serial: "V4-TEST"},
	}
}

// TestDeviceSetBlogV4FlipsCrystal is the issue #264 regression at the
// Device layer: forcing Blog V4 mode must swap the R828D's reference
// crystal from the generic 16 MHz default to 28.8 MHz, which is what
// stops the LO mistuning by ~1.8×.
func TestDeviceSetBlogV4FlipsCrystal(t *testing.T) {
	d := newDeviceWithR828D(t)

	// Before: an undetected R828D sits on the 16 MHz generic crystal —
	// the mistune signature.
	name, v4, lite, xtal := d.TunerDiag()
	if name != "R828D" || v4 || lite || xtal != 16_000_000 {
		t.Fatalf("pre-force TunerDiag = (%q, %v, %v, %d), want (\"R828D\", false, false, 16000000)",
			name, v4, lite, xtal)
	}

	if err := d.SetBlogV4(false); err != nil {
		t.Fatalf("SetBlogV4: %v", err)
	}

	// After: 28.8 MHz crystal and the V4 path armed.
	name, v4, lite, xtal = d.TunerDiag()
	if name != "R828D" || !v4 || lite || xtal != 28_800_000 {
		t.Fatalf("post-force TunerDiag = (%q, %v, %v, %d), want (\"R828D\", true, false, 28800000)",
			name, v4, lite, xtal)
	}
}

// TestDeviceSetBlogV4Lite confirms the Lite flag reaches the tuner.
func TestDeviceSetBlogV4Lite(t *testing.T) {
	d := newDeviceWithR828D(t)
	if err := d.SetBlogV4(true); err != nil {
		t.Fatalf("SetBlogV4: %v", err)
	}
	if _, v4, lite, _ := d.TunerDiag(); !v4 || !lite {
		t.Fatalf("after SetBlogV4(true): v4=%v lite=%v, want both true", v4, lite)
	}
}

// TestDeviceSetBlogV4NonR82xxNoop verifies the force is a safe no-op on a
// tuner that isn't an R82xx (so the optional interface can be wired
// unconditionally in the pool without guarding on chip type).
func TestDeviceSetBlogV4NonR82xxNoop(t *testing.T) {
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), &fakeTuner{})
	if err := d.SetBlogV4(false); err != nil {
		t.Fatalf("SetBlogV4 on non-R82xx tuner: %v", err)
	}
	// TunerDiag falls back to the chip name with no crystal info.
	name, v4, lite, xtal := d.TunerDiag()
	if v4 || lite || xtal != 0 {
		t.Fatalf("non-R82xx TunerDiag = (%q, %v, %v, %d), want (..., false, false, 0)",
			name, v4, lite, xtal)
	}
}

// TestDeviceSetBlogV4Closed guards the closed path.
func TestDeviceSetBlogV4Closed(t *testing.T) {
	d := newDeviceWithR828D(t)
	d.closed.Store(true)
	if err := d.SetBlogV4(false); err != ErrClosed {
		t.Fatalf("SetBlogV4 on closed device = %v, want ErrClosed", err)
	}
}
