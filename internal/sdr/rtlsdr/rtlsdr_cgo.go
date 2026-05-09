// Package rtlsdr is a thin CGO binding to librtlsdr. It exposes the subset
// of the C API required by GopherTrunk: enumeration, tuning, sample rate,
// gain, PPM, and an async read loop bridged to a Go channel of complex64.
//
// The build links against -lrtlsdr and requires librtlsdr-dev headers on
// the build host. Runtime needs udev rules granting USB access (see
// docs/hardware.md).
package rtlsdr

/*
#cgo pkg-config: librtlsdr
#include <rtl-sdr.h>
#include <stdint.h>
#include <stdlib.h>

extern void gophertrunk_rtlsdr_callback(unsigned char *buf, uint32_t len, void *ctx);

// Wrapper that accepts the cgo.Handle as uintptr_t; this avoids the
// uintptr->unsafe.Pointer conversion vet warning on the Go side.
static int gt_rtlsdr_read_async(rtlsdr_dev_t *dev, uintptr_t handle,
                                uint32_t buf_num, uint32_t buf_len) {
    return rtlsdr_read_async(dev, gophertrunk_rtlsdr_callback,
                             (void *)handle, buf_num, buf_len);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime/cgo"
	"sync"
	"unsafe"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

const driverName = "rtlsdr"

// Default async-buffer geometry. 32 buffers × 16 KiB → ~6 ms at 2.4 MS/s.
const (
	asyncBufCount = 32
	asyncBufLen   = 16 * 1024
)

func init() { sdr.Register(driver{}) }

type driver struct{}

func (driver) Name() string { return driverName }

func (driver) Enumerate() ([]sdr.Info, error) {
	count := int(C.rtlsdr_get_device_count())
	out := make([]sdr.Info, 0, count)
	for i := 0; i < count; i++ {
		var manuf, prod, ser [256]C.char
		C.rtlsdr_get_device_usb_strings(C.uint32_t(i), &manuf[0], &prod[0], &ser[0])
		info := sdr.Info{
			Driver:       driverName,
			Index:        i,
			Manufacturer: C.GoString(&manuf[0]),
			Product:      C.GoString(&prod[0]),
			Serial:       C.GoString(&ser[0]),
		}
		out = append(out, info)
	}
	return out, nil
}

func (driver) Open(idx int) (sdr.Device, error) {
	var dev *C.rtlsdr_dev_t
	if rc := C.rtlsdr_open(&dev, C.uint32_t(idx)); rc != 0 {
		return nil, fmt.Errorf("rtlsdr_open(%d): rc=%d", idx, rc)
	}
	d := &Device{dev: dev, index: idx}
	d.populateInfo()
	d.populateGains()
	return d, nil
}

// Device is a handle to one RTL-SDR dongle.
type Device struct {
	mu     sync.Mutex
	dev    *C.rtlsdr_dev_t
	index  int
	info   sdr.Info
	closed bool

	// streaming state, owned while StreamIQ is running.
	streamMu sync.Mutex
	handle   cgo.Handle
	out      chan []complex64
	stopOnce sync.Once
}

func (d *Device) populateInfo() {
	d.info = sdr.Info{Driver: driverName, Index: d.index}
	var manuf, prod, ser [256]C.char
	C.rtlsdr_get_usb_strings(d.dev, &manuf[0], &prod[0], &ser[0])
	d.info.Manufacturer = C.GoString(&manuf[0])
	d.info.Product = C.GoString(&prod[0])
	d.info.Serial = C.GoString(&ser[0])
	tuner := C.rtlsdr_get_tuner_type(d.dev)
	d.info.TunerName = tunerName(tuner)
}

func (d *Device) populateGains() {
	n := int(C.rtlsdr_get_tuner_gains(d.dev, nil))
	if n <= 0 {
		return
	}
	buf := make([]C.int, n)
	C.rtlsdr_get_tuner_gains(d.dev, (*C.int)(unsafe.Pointer(&buf[0])))
	d.info.Gains = make([]int, n)
	for i, g := range buf {
		d.info.Gains[i] = int(g)
	}
}

func (d *Device) Info() sdr.Info { return d.info }

func (d *Device) SetCenterFreq(hz uint32) error {
	if rc := C.rtlsdr_set_center_freq(d.dev, C.uint32_t(hz)); rc != 0 {
		return fmt.Errorf("rtlsdr_set_center_freq(%d): rc=%d", hz, rc)
	}
	return nil
}

func (d *Device) SetSampleRate(hz uint32) error {
	if rc := C.rtlsdr_set_sample_rate(d.dev, C.uint32_t(hz)); rc != 0 {
		return fmt.Errorf("rtlsdr_set_sample_rate(%d): rc=%d", hz, rc)
	}
	return nil
}

func (d *Device) SetGain(tenthDB int) error {
	if tenthDB < 0 {
		if rc := C.rtlsdr_set_tuner_gain_mode(d.dev, 0); rc != 0 {
			return fmt.Errorf("rtlsdr_set_tuner_gain_mode(auto): rc=%d", rc)
		}
		return nil
	}
	if rc := C.rtlsdr_set_tuner_gain_mode(d.dev, 1); rc != 0 {
		return fmt.Errorf("rtlsdr_set_tuner_gain_mode(manual): rc=%d", rc)
	}
	if rc := C.rtlsdr_set_tuner_gain(d.dev, C.int(tenthDB)); rc != 0 {
		return fmt.Errorf("rtlsdr_set_tuner_gain(%d): rc=%d", tenthDB, rc)
	}
	return nil
}

func (d *Device) SetPPM(ppm int) error {
	if rc := C.rtlsdr_set_freq_correction(d.dev, C.int(ppm)); rc != 0 && rc != -2 {
		// -2 means "value unchanged" on librtlsdr; treat as success.
		return fmt.Errorf("rtlsdr_set_freq_correction(%d): rc=%d", ppm, rc)
	}
	return nil
}

// SetBiasTee toggles the dongle's software-controlled bias-tee, the
// 5V output that powers external LNAs through the antenna SMA. The
// NooElec NESDR Smart v5 and most modern RTL-SDR clones expose this;
// older units route the GPIO to a missing component (the call still
// succeeds but no voltage appears). librtlsdr handles the GPIO bit
// regardless of tuner type.
func (d *Device) SetBiasTee(enable bool) error {
	on := C.int(0)
	if enable {
		on = 1
	}
	if rc := C.rtlsdr_set_bias_tee(d.dev, on); rc != 0 {
		return fmt.Errorf("rtlsdr_set_bias_tee(%v): rc=%d", enable, rc)
	}
	return nil
}

// StreamIQ resets the buffer, kicks off rtlsdr_read_async on a goroutine, and
// returns a channel of complex64 samples. The channel closes when ctx cancels
// or Close is called.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.streamMu.Lock()
	defer d.streamMu.Unlock()
	if d.out != nil {
		return nil, errors.New("rtlsdr: stream already active")
	}
	if rc := C.rtlsdr_reset_buffer(d.dev); rc != 0 {
		return nil, fmt.Errorf("rtlsdr_reset_buffer: rc=%d", rc)
	}
	out := make(chan []complex64, 8)
	d.out = out
	d.handle = cgo.NewHandle(d)
	d.stopOnce = sync.Once{}

	go func() {
		C.gt_rtlsdr_read_async(
			d.dev,
			C.uintptr_t(d.handle),
			C.uint32_t(asyncBufCount),
			C.uint32_t(asyncBufLen),
		)
		d.streamMu.Lock()
		if d.out != nil {
			close(d.out)
			d.out = nil
		}
		d.handle.Delete()
		d.streamMu.Unlock()
	}()

	go func() {
		<-ctx.Done()
		d.cancelStream()
	}()

	return out, nil
}

func (d *Device) cancelStream() {
	d.stopOnce.Do(func() {
		C.rtlsdr_cancel_async(d.dev)
	})
}

func (d *Device) deliver(buf []byte) {
	// RTL-SDR delivers unsigned 8-bit IQ pairs. Convert to complex64 with a
	// DC bias of 127.5 → range ~[-1, +1).
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		i8 := float32(buf[2*i]) - 127.5
		q8 := float32(buf[2*i+1]) - 127.5
		out[i] = complex(i8/127.5, q8/127.5)
	}
	select {
	case d.out <- out:
	default:
		// Drop on overrun; consumer is too slow.
	}
}

func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	d.cancelStream()
	if rc := C.rtlsdr_close(d.dev); rc != 0 {
		return fmt.Errorf("rtlsdr_close: rc=%d", rc)
	}
	return nil
}

func tunerName(t C.enum_rtlsdr_tuner) string {
	switch t {
	case C.RTLSDR_TUNER_E4000:
		return "E4000"
	case C.RTLSDR_TUNER_FC0012:
		return "FC0012"
	case C.RTLSDR_TUNER_FC0013:
		return "FC0013"
	case C.RTLSDR_TUNER_FC2580:
		return "FC2580"
	case C.RTLSDR_TUNER_R820T:
		return "R820T"
	case C.RTLSDR_TUNER_R828D:
		return "R828D"
	default:
		return "unknown"
	}
}

//export gophertrunk_rtlsdr_callback
func gophertrunk_rtlsdr_callback(buf *C.uchar, length C.uint32_t, ctx unsafe.Pointer) {
	h := cgo.Handle(uintptr(ctx))
	d, ok := h.Value().(*Device)
	if !ok || d == nil {
		return
	}
	// Copy the C buffer into a Go slice before returning; the callback
	// reuses the buffer immediately on return.
	n := int(length)
	src := unsafe.Slice((*byte)(unsafe.Pointer(buf)), n)
	cp := make([]byte, n)
	copy(cp, src)
	d.deliver(cp)
}
