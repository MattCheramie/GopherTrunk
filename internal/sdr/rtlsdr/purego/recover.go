package purego

import (
	"errors"
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// Runtime recovery from a wedged control pipe (issue #753).
//
// The reporter's RTL-SDR Blog V4 on a Raspberry Pi 5 ran the conventional
// scanner until the first retune after a dwell, then EVERY control transfer
// failed with EPIPE until the dongle was replugged. The demod's in-place
// settle-and-retry (rtl2832u.EnableControlStallRetry) cannot help there: it
// retries the same write 5 ms later on a pipe that stays stalled. Two
// changes, both confined to the window where no IQ stream is running:
//
//  1. Control methods wait for a cancelled stream's teardown to finish
//     before touching the control pipe. The conventional scanner cancels a
//     dwell's stream and retunes at once, and the teardown (StopBulkIn
//     discarding and reaping every in-flight URB) runs on its own
//     goroutine, so the retune's control writes used to race it. The scanner
//     is the only caller that stops and retunes back to back; fixed-tune
//     TETRA, rtl_test and SpyServer (all fine on the same Pi 5) never do.
//  2. A control transfer that still stalls (or times out) with no stream
//     running gets one USB reset plus the open-time bring-up, i.e. the
//     replug the reporter does by hand, after which the device's last
//     programmed state is restored and the call retried once. Never while a
//     stream is live: a reset would tear the stream down mid-call.

// teardownWait bounds how long a control call waits for a cancelled stream's
// teardown. StopBulkIn normally finishes in milliseconds; the bound only
// keeps a wedged reaper from hanging the caller.
const teardownWait = 2 * time.Second

// controlRecoveryCooldown is the minimum gap between two recoveries, so a
// dongle that is genuinely dead (or a reset that does not help) is retried
// at a bounded rate instead of reset-storming its host controller.
const controlRecoveryCooldown = 30 * time.Second

// controlRecoverySettle is the pause between the USB reset and re-claiming
// the interface, as in the open-time envelope.
var controlRecoverySettle = 200 * time.Millisecond

// errRecoveryUnavailable reports why no recovery ran; it is folded into the
// returned error's text, never matched.
var errRecoveryUnavailable = errors.New("recovery unavailable")

// appliedState is what the caller has programmed into the device, kept so a
// re-bring-up can restore it. Guarded by Device.hwMu.
type appliedState struct {
	rateHz   uint32
	freqHz   uint32
	ppm      int
	ppmSet   bool
	gain     int
	gainSet  bool
	biasTee  bool
	biasSet  bool
	blogV4   bool
	blogLite bool
}

// isRecoverableControlErr reports whether err is a control-pipe failure a USB
// reset can clear: a STALL (EPIPE on Linux, ErrPipeStalled on Windows/macOS)
// or a timeout (the R820T2's -110 on the same Pi 5).
func isRecoverableControlErr(err error) bool {
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, usb.ErrPipeStalled) ||
		errors.Is(err, usb.ErrTimeout)
}

// awaitStreamTeardown blocks while a stream whose ctx has been cancelled is
// still being torn down. A live stream (ctx not cancelled) is not waited
// for: control transfers during streaming are normal (retunes under a
// trunking decoder). Must not be called with hwMu held (lock order is
// streamMu before hwMu).
func (d *Device) awaitStreamTeardown() {
	d.streamMu.Lock()
	done := d.streamDone
	cancelled := d.out != nil && d.streamCtx != nil && d.streamCtx.Err() != nil
	d.streamMu.Unlock()
	if !cancelled || done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(teardownWait):
	}
}

// control runs one control-path operation: after any pending stream
// teardown, under hwMu, with one reset-and-rebring-up retry on a recoverable
// failure while no stream is running. op must read d.demod / d.tuner itself
// (not captured copies) so the retry uses the re-opened chip.
func (d *Device) control(op func() error) error {
	if d.closed.Load() {
		return ErrClosed
	}
	d.awaitStreamTeardown()
	d.hwMu.Lock()
	defer d.hwMu.Unlock()
	err := op()
	if err == nil || !isRecoverableControlErr(err) {
		return err
	}
	if rerr := d.recoverLocked(err); rerr != nil {
		return fmt.Errorf("%w (USB reset recovery: %v)", err, rerr)
	}
	return op()
}

// recoverLocked resets the USB device, re-runs the open-time bring-up and
// restores the applied state. Caller holds hwMu.
func (d *Device) recoverLocked(cause error) error {
	switch {
	case d.rebringup == nil:
		return fmt.Errorf("%w: device has no bring-up hook", errRecoveryUnavailable)
	case d.streaming.Load():
		return fmt.Errorf("%w: an IQ stream is running", errRecoveryUnavailable)
	case !d.lastRecovery.IsZero() && time.Since(d.lastRecovery) < controlRecoveryCooldown:
		return fmt.Errorf("%w: last reset %s ago", errRecoveryUnavailable, time.Since(d.lastRecovery).Round(time.Second))
	}
	d.lastRecovery = time.Now()
	d.recoveries.Add(1)
	slog.Warn("rtlsdr: control transfer failed with no IQ stream running; resetting the dongle and re-running bring-up (issue #753)",
		"serial", d.info.Serial, "err", cause)

	if err := d.transport.Reset(); err != nil {
		return fmt.Errorf("USB reset: %w", err)
	}
	time.Sleep(controlRecoverySettle)
	_ = d.transport.ReleaseInterface(0)
	if err := d.transport.ClaimInterface(0); err != nil {
		return fmt.Errorf("re-claim interface 0 after reset: %w", err)
	}
	demod, tuner, err := d.rebringup()
	if err != nil {
		return fmt.Errorf("bring-up after reset: %w", err)
	}
	d.demod, d.tuner = demod, tuner
	if err := d.restoreLocked(); err != nil {
		return fmt.Errorf("restore settings after reset: %w", err)
	}
	slog.Info("rtlsdr: dongle recovered after USB reset", "serial", d.info.Serial)
	return nil
}

// restoreLocked re-programs the applied state onto a freshly brought-up
// chip, in open order: V4 mode, sample rate, ppm, gain, bias tee, frequency.
func (d *Device) restoreLocked() error {
	a := d.applied
	if a.blogV4 {
		if r82, ok := d.tuner.(*tuners.R82xx); ok {
			r82.SetBlogV4(a.blogLite)
		}
	}
	if a.rateHz != 0 {
		if err := d.setSampleRateLocked(a.rateHz); err != nil {
			return err
		}
	}
	if a.ppmSet {
		if err := d.setPPMLocked(a.ppm); err != nil {
			return err
		}
	}
	if a.gainSet {
		if err := d.setGainLocked(a.gain); err != nil {
			return err
		}
	}
	if a.biasSet {
		if err := d.demod.SetBiasTee(d.biasTeeGPIO, a.biasTee); err != nil {
			return err
		}
	}
	if a.freqHz != 0 {
		if err := d.tuner.SetFreq(a.freqHz); err != nil {
			return err
		}
	}
	return nil
}

// ControlRecoveries reports how many reset-and-rebring-up recoveries this
// device has run.
func (d *Device) ControlRecoveries() uint64 { return d.recoveries.Load() }

// newRebringup is the recovery hook openDeviceAttempts installs: the same
// bring-up an Open runs, on the same transport, with the V4 detection
// re-armed from the descriptor strings.
func newRebringup(transport usb.Transport, manufacturer, product string) func() (*rtl2832u.Demod, tuners.Tuner, error) {
	return func() (*rtl2832u.Demod, tuners.Tuner, error) {
		demod := rtl2832u.New(transport)
		tuner, err := runBringup(demod)
		if err != nil {
			return nil, nil, err
		}
		demod.EnableControlStallRetry()
		armBlogV4FromDescriptor(tuner, manufacturer, product)
		return demod, tuner, nil
	}
}

// armBlogV4FromDescriptor arms the R828D's V4 path when the USB strings name
// an RTL-SDR Blog V4 (issue #264).
func armBlogV4FromDescriptor(tuner tuners.Tuner, manufacturer, product string) {
	if r82, ok := tuner.(*tuners.R82xx); ok {
		if lite, isV4 := blogV4Variant(manufacturer, product); isV4 {
			r82.SetBlogV4(lite)
		}
	}
}
