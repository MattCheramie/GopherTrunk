package rtl2832u

import (
	"errors"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// controlStallSettle is the pause before retrying a control transfer that
// stalled the pipe, giving the device firmware and the host USB stack a
// moment to clear the STALL — the same recovery libusb performs implicitly,
// which is why rtl_test / SDR++ tune a dongle that stalls GopherTrunk.
const controlStallSettle = 5 * time.Millisecond

// EnableControlStallRetry arms runtime control-pipe-stall recovery on the
// demod / block / I2C control transfers. Call it once, AFTER bring-up
// completes.
//
// During bring-up a control-pipe stall is the cold-boot latch that only a
// full device reset clears, and the open-time retry envelope
// (purego.runBringup) owns that — so the settle-and-retry here must stay off
// until bring-up is done, or it would mask the stall the reset needs to see.
// Once armed, an intermittent RUNTIME stall — most visibly the SetI2CRepeater
// toggle inside a SetCenterFreq retune on the RTL-SDR Blog V4's R828D
// (issue #753) — is settled and retried once in place instead of aborting the
// whole tune. A full device reset is deliberately NOT used at runtime: it
// would tear down the live IQ stream mid-tune.
func (d *Demod) EnableControlStallRetry() {
	d.mu.Lock()
	d.stallRetry = true
	d.mu.Unlock()
}

// isControlPipeStall reports whether err is a recoverable control-pipe STALL:
// raw syscall.EPIPE on Linux/usbdevfs, usb.ErrPipeStalled on Windows/WinUSB
// and macOS/IOKit.
func isControlPipeStall(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, usb.ErrPipeStalled)
}

// ctrlOut issues a vendor-OUT control transfer, settling and retrying ONCE on
// a control-pipe stall when runtime recovery is armed (EnableControlStallRetry).
// Callers hold d.mu. A stall that survives the single retry still surfaces, so
// a genuinely wedged pipe is never hidden by silent looping.
func (d *Demod) ctrlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	err := d.t.ControlOut(bRequest, wValue, wIndex, data, timeoutMs)
	if err != nil && d.stallRetry && isControlPipeStall(err) {
		time.Sleep(controlStallSettle)
		err = d.t.ControlOut(bRequest, wValue, wIndex, data, timeoutMs)
	}
	return err
}

// ctrlIn is the vendor-IN counterpart of ctrlOut, with the same
// settle-and-retry-once recovery. Callers hold d.mu.
func (d *Demod) ctrlIn(bRequest uint8, wValue, wIndex uint16, n, timeoutMs int) ([]byte, error) {
	out, err := d.t.ControlIn(bRequest, wValue, wIndex, n, timeoutMs)
	if err != nil && d.stallRetry && isControlPipeStall(err) {
		time.Sleep(controlStallSettle)
		out, err = d.t.ControlIn(bRequest, wValue, wIndex, n, timeoutMs)
	}
	return out, err
}
