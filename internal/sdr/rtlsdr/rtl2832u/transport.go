package rtl2832u

import (
	"fmt"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

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
	out, err := d.t.ControlIn(0, addr, index, n, CtrlTimeoutMs)
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
	if err := d.t.ControlOut(0, addr, index, data, CtrlTimeoutMs); err != nil {
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
	out, err := d.t.ControlIn(0, wValue, wIndex, n, CtrlTimeoutMs)
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
	if err := d.t.ControlOut(0, wValue, wIndex, data, CtrlTimeoutMs); err != nil {
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
