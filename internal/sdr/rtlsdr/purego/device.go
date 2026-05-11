package purego

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// biasTeeGPIO is the GPIO pin that drives the 5 V LNA-power output on
// the dominant RTL-SDR designs (RTL-SDR.com v3+, NESDR Smart v5).
// Other clones map bias-tee to different pins; future work can table
// this per (VID, PID) — see the per-product mapping comment in
// rtl2832u.SetBiasTee.
const biasTeeGPIO uint8 = 0

// ErrClosed is returned by [Device] methods invoked after [Device.Close].
var ErrClosed = errors.New("rtlsdr-go: device closed")

// Device implements [sdr.Device] for the pure-Go backend. It composes
// the platform USB transport, the RTL2832U register layer, and the
// per-chip tuner driver. Methods are goroutine-safe by virtue of the
// underlying components being safe under their own mutexes; Device
// adds light coordination for the streaming-state machine and the
// closed flag.
type Device struct {
	transport usb.Transport
	demod     *rtl2832u.Demod
	tuner     tuners.Tuner
	info      sdr.Info

	streamMu sync.Mutex
	out      chan []complex64
	stopOnce sync.Once

	closed atomic.Bool
}

// Info returns the descriptor populated by [Driver.Open] — driver
// name, USB index, serial / manufacturer / product strings, tuner
// chip name, and the tuner's gain ladder.
func (d *Device) Info() sdr.Info { return d.info }

// SetCenterFreq tunes the LO. Delegates to the tuner; settles
// briefly afterwards so the PLL has time to lock.
func (d *Device) SetCenterFreq(hz uint32) error {
	if d.closed.Load() {
		return ErrClosed
	}
	if err := d.tuner.SetFreq(hz); err != nil {
		return fmt.Errorf("rtlsdr-go: SetCenterFreq(%d): %w", hz, err)
	}
	if r, ok := d.tuner.(interface{ SettleAfterRetune() }); ok {
		r.SettleAfterRetune()
	}
	return nil
}

// SetSampleRate programs the demod resampler and the tuner's IF
// filter to match. Returns the actual rate the chip can produce
// (the demod quantizes to its 28.4 fixed-point divisor); callers
// that need the exact value can read it back via [Demod.GetSampleRate]
// — but the [sdr.Device] contract returns just the error.
func (d *Device) SetSampleRate(hz uint32) error {
	if d.closed.Load() {
		return ErrClosed
	}
	actual, err := d.demod.SetSampleRate(hz)
	if err != nil {
		return fmt.Errorf("rtlsdr-go: SetSampleRate(%d): %w", hz, err)
	}
	if err := d.tuner.SetBandwidth(actual); err != nil {
		return fmt.Errorf("rtlsdr-go: tuner bandwidth: %w", err)
	}
	return nil
}

// SetGain takes the [sdr.Device] convention: tenthDB < 0 selects AGC,
// tenthDB ≥ 0 switches to manual mode and applies the closest
// quantized gain on the chip's ladder. Mirrors librtlsdr's
// rtlsdr_set_tuner_gain_mode + rtlsdr_set_tuner_gain pair.
func (d *Device) SetGain(tenthDB int) error {
	if d.closed.Load() {
		return ErrClosed
	}
	if tenthDB < 0 {
		if err := d.tuner.SetGainMode(false); err != nil {
			return fmt.Errorf("rtlsdr-go: SetGain auto: %w", err)
		}
		return nil
	}
	if err := d.tuner.SetGainMode(true); err != nil {
		return fmt.Errorf("rtlsdr-go: SetGain manual mode: %w", err)
	}
	if err := d.tuner.SetGain(tenthDB); err != nil {
		return fmt.Errorf("rtlsdr-go: SetGain(%d tenthDB): %w", tenthDB, err)
	}
	return nil
}

// SetPPM applies a parts-per-million correction to the resampler.
// Negative slows the chip; positive speeds it up.
func (d *Device) SetPPM(ppm int) error {
	if d.closed.Load() {
		return ErrClosed
	}
	if err := d.demod.SetSampleFreqCorrection(ppm); err != nil {
		return fmt.Errorf("rtlsdr-go: SetPPM(%d): %w", ppm, err)
	}
	return nil
}

// SetBiasTee toggles the dongle's 5 V LNA-power output. The GPIO pin
// is hard-coded to 0 — the standard for RTL-SDR.com v3+ and NESDR
// Smart v5 dongles. Boards that wire bias-tee to a different pin
// will silently no-op; per-product pin mapping is a future
// enhancement (see the comment in rtl2832u.SetBiasTee).
func (d *Device) SetBiasTee(enable bool) error {
	if d.closed.Load() {
		return ErrClosed
	}
	if err := d.demod.SetBiasTee(biasTeeGPIO, enable); err != nil {
		return fmt.Errorf("rtlsdr-go: SetBiasTee(%v): %w", enable, err)
	}
	return nil
}

// Close releases the tuner, powers off the demod, and closes the USB
// transport. Idempotent; safe to call from any goroutine.
func (d *Device) Close() error {
	if !d.closed.CompareAndSwap(false, true) {
		return nil
	}
	d.cancelStream()
	var firstErr error
	if d.tuner != nil {
		if err := d.tuner.Standby(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("rtlsdr-go: tuner standby: %w", err)
		}
	}
	if d.demod != nil {
		if err := d.demod.DeinitBaseband(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("rtlsdr-go: demod deinit: %w", err)
		}
	}
	if err := d.transport.Close(); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("rtlsdr-go: transport close: %w", err)
	}
	return firstErr
}

// cancelStream is the path closed-channel-on-error and ctx-cancel both
// take. Idempotent via stopOnce — stops the bulk-IN ring and closes
// the consumer channel.
func (d *Device) cancelStream() {
	d.stopOnce.Do(func() {
		_ = d.transport.StopBulkIn()
		d.streamMu.Lock()
		if d.out != nil {
			close(d.out)
			d.out = nil
		}
		d.streamMu.Unlock()
	})
}

// StreamIQ is implemented in stream.go (kept separate so the IQ
// conversion math sits next to its unit test).
//
// Compile-time assertion that Device satisfies sdr.Device.
var _ sdr.Device = (*Device)(nil)

// ensure the unused-import warning stays away when the build skips
// the streaming code path (it doesn't today; this is a guard for the
// future).
var _ = context.TODO
