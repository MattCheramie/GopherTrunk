package rtl2832u

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// ctrlStallRetryDelay is the settle applied before retrying a demod control
// transfer that STALLed the USB control pipe. A package var (not a const) so
// tests can zero it. Matches the tuner burst path's r82xxBurstRetryDelayMillis
// order of magnitude — long enough to let the RTL2832U's I²C bridge drain,
// short enough to be invisible on a healthy retune.
var ctrlStallRetryDelay = 8 * time.Millisecond

// isControlPipeStall reports whether err is a recoverable USB control-pipe
// stall: the RTL-SDR Blog V4's R828D intermittently STALLs a demod control
// write (most visibly the SetI2CRepeater toggle inside a retune), which
// surfaces as a raw syscall.EPIPE on Linux usbdevfs or usb.ErrPipeStalled on
// Windows/WinUSB. Mirrors the tuner path's isI2CBurstStall predicate. A
// control-endpoint stall is cleared by the next SETUP, so a single resubmit
// recovers it. Issue #753.
func isControlPipeStall(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, usb.ErrPipeStalled)
}

// ctrlOut issues a vendor-OUT demod/block control transfer, retrying once on a
// recoverable control-pipe stall (isControlPipeStall). librtlsdr/libusb get
// this recovery for free — which is why rtl_test / SDR++ retune cleanly on the
// same RTL-SDR Blog V4 where GopherTrunk's native stack surfaced the stall as
// `SetI2CRepeater(true): ... broken pipe` and aborted the whole SetCenterFreq
// (issue #753). A full device Reset (the open-time recovery envelope) is
// deliberately NOT used on this runtime path: it would tear down the live IQ
// stream mid-tune. The register writes routed through here carry absolute
// values, so replaying a stalled (un-applied) write is idempotent.
func (d *Demod) ctrlOut(wValue, wIndex uint16, data []byte) error {
	err := d.t.ControlOut(0, wValue, wIndex, data, CtrlTimeoutMs)
	if err == nil || !d.stallRetry || !isControlPipeStall(err) {
		return err
	}
	time.Sleep(ctrlStallRetryDelay)
	return d.t.ControlOut(0, wValue, wIndex, data, CtrlTimeoutMs)
}

// ctrlIn is the vendor-IN counterpart to ctrlOut, applying the same
// single-retry recovery to a stalled demod/block register read. Issue #753.
func (d *Demod) ctrlIn(wValue, wIndex uint16, n int) ([]byte, error) {
	out, err := d.t.ControlIn(0, wValue, wIndex, n, CtrlTimeoutMs)
	if err == nil || !d.stallRetry || !isControlPipeStall(err) {
		return out, err
	}
	time.Sleep(ctrlStallRetryDelay)
	return d.t.ControlIn(0, wValue, wIndex, n, CtrlTimeoutMs)
}

// Demod is the per-device register interface. One [Demod] wraps one
// claimed [usb.Transport] and is owned by the higher-level driver
// (PR-06). It serializes control transfers behind a mutex so concurrent
// register accesses from multiple goroutines (e.g. tuner code racing
// with bias-tee toggling) interleave cleanly on the USB wire.
//
// All ReadBlockReg / WriteBlockReg / ReadDemodReg / WriteDemodReg
// methods produce control transfers byte-identical to what librtlsdr
// sends — confirmed by the golden tests in this package.
type Demod struct {
	t     usb.Transport
	mu    sync.Mutex
	xtal  uint32
	rate  uint32
	ifHz  int32
	ppm   int
	repON bool // last value pushed to SetI2CRepeater
	// stallRetry enables the single settle-and-retry recovery in ctrlOut /
	// ctrlIn on a control-pipe stall. OFF during open-time bring-up, where a
	// cold-boot stall must propagate to the driver's reset+retry envelope (a
	// full USBDEVFS_RESET is the only thing that clears the NESDR-v5
	// latch-and-NAK; an inline resubmit just re-NAKs, issue #248). Enabled by
	// the driver via EnableControlStallRetry once bring-up succeeds, so the
	// runtime retune path recovers a transient stall without a stream-killing
	// reset (issue #753).
	stallRetry bool
}

// EnableControlStallRetry turns on the runtime control-pipe stall recovery in
// ctrlOut / ctrlIn. The driver calls it after a successful bring-up; see the
// stallRetry field. Issue #753.
func (d *Demod) EnableControlStallRetry() {
	d.mu.Lock()
	d.stallRetry = true
	d.mu.Unlock()
}

// New wraps the given transport. The caller is responsible for opening
// and claiming the device first; this layer doesn't manage the USB
// handle, only the protocol on top of it.
func New(t usb.Transport) *Demod {
	return &Demod{t: t, xtal: DefaultXtalHz}
}

// SetXtal overrides the default 28.8 MHz reference frequency for boards
// that ship with a non-standard crystal. Sample-rate and IF math both
// scale against this value.
func (d *Demod) SetXtal(hz uint32) {
	d.mu.Lock()
	d.xtal = hz
	d.mu.Unlock()
}

// XtalHz returns the current reference-crystal value in Hz.
func (d *Demod) XtalHz() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.xtal
}

// GetSampleRate returns the most-recently-programmed sample rate in Hz,
// adjusted for the chip's actual resampler resolution (not the value
// the caller requested).
func (d *Demod) GetSampleRate() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rate
}

// ReadBlockReg issues a vendor-IN control transfer against the given
// block at the given address, returning n bytes. Matches librtlsdr's
// rtlsdr_read_reg: wValue=addr, wIndex=block<<8, bmRequestType=0xC0.
//
// On the wire, the chip returns bytes in little-endian for n=2 reads.
// Callers wanting a uint16 should mask/shift accordingly (or use the
// ReadBlockRegU16 convenience).
func (d *Demod) ReadBlockReg(block uint8, addr uint16, n int) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readBlockRegLocked(block, addr, n)
}

func (d *Demod) readBlockRegLocked(block uint8, addr uint16, n int) ([]byte, error) {
	index := uint16(block) << 8
	out, err := d.ctrlIn(addr, index, n)
	if err != nil {
		return nil, fmt.Errorf("rtl2832u: read block=%d addr=0x%04x: %w", block, addr, err)
	}
	return out, nil
}

// WriteBlockReg writes a register in the given block. Encoding matches
// rtlsdr_write_reg: wValue=addr, wIndex=block<<8|0x10. For n=1 the low
// byte of val is sent; for n=2 val is sent big-endian (data[0]=high,
// data[1]=low).
func (d *Demod) WriteBlockReg(block uint8, addr, val uint16, n int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeBlockRegLocked(block, addr, val, n)
}

func (d *Demod) writeBlockRegLocked(block uint8, addr, val uint16, n int) error {
	index := uint16(block)<<8 | 0x10
	data := encodeWriteVal(val, n)
	if err := d.ctrlOut(addr, index, data); err != nil {
		return fmt.Errorf("rtl2832u: write block=%d addr=0x%04x val=0x%04x: %w", block, addr, val, err)
	}
	return nil
}

// ReadDemodReg reads from the page-addressed demod register space.
// Matches rtlsdr_demod_read_reg: wValue=(addr<<8)|0x20, wIndex=page,
// bmRequestType=0xC0.
//
// Callers usually want one byte; on n=2 the chip emits little-endian.
func (d *Demod) ReadDemodReg(page uint8, addr uint16, n int) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readDemodRegLocked(page, addr, n)
}

func (d *Demod) readDemodRegLocked(page uint8, addr uint16, n int) ([]byte, error) {
	wValue := (addr << 8) | 0x20
	wIndex := uint16(page)
	out, err := d.ctrlIn(wValue, wIndex, n)
	if err != nil {
		return nil, fmt.Errorf("rtl2832u: read demod page=%d addr=0x%02x: %w", page, addr, err)
	}
	return out, nil
}

// WriteDemodReg writes to the page-addressed demod register space.
// Matches rtlsdr_demod_write_reg: wValue=(addr<<8)|0x20,
// wIndex=0x10|page, bmRequestType=0x40. Each write is followed by a
// "commit" read of page 0x0A register 0x01 — without this dummy read
// the chip does not latch the previous write. Match the C library bit
// for bit; the commit read is load-bearing on real hardware.
func (d *Demod) WriteDemodReg(page uint8, addr, val uint16, n int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeDemodRegLocked(page, addr, val, n)
}

func (d *Demod) writeDemodRegLocked(page uint8, addr, val uint16, n int) error {
	wValue := (addr << 8) | 0x20
	wIndex := uint16(0x10) | uint16(page)
	data := encodeWriteVal(val, n)
	if err := d.ctrlOut(wValue, wIndex, data); err != nil {
		return fmt.Errorf("rtl2832u: write demod page=%d addr=0x%02x val=0x%04x: %w", page, addr, val, err)
	}
	// Commit. Required by the RTL2832U register interface — without it
	// the demod write doesn't take effect. Errors here are non-fatal
	// (the underlying write already happened), so we swallow them.
	_, _ = d.readDemodRegLocked(0x0A, 0x01, 1)
	return nil
}

// encodeWriteVal mirrors the C library's data layout: 1-byte writes
// send val & 0xff; 2-byte writes send big-endian (high byte first).
// Anything else is a programmer error and we panic — the package
// never has cause to issue an n=0 or n>2 write.
func encodeWriteVal(val uint16, n int) []byte {
	switch n {
	case 1:
		return []byte{byte(val & 0xff)}
	case 2:
		return []byte{byte(val >> 8), byte(val & 0xff)}
	default:
		panic(fmt.Sprintf("rtl2832u: encodeWriteVal n=%d not in {1,2}", n))
	}
}
