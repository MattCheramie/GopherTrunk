package purego

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Async-bulk geometry copied verbatim from the CGO driver
// (internal/sdr/rtlsdr/rtlsdr_cgo.go:42-45). 32 buffers × 16 KiB ≈
// 6 ms of headroom at 2.4 MS/s. Changing these requires a
// matching update on the C side until PR-09 deletes that file.
const (
	asyncBufCount = 32
	asyncBufLen   = 16 * 1024

	// streamChanDepth matches rtlsdr_cgo.go:190 — `make(chan
	// []complex64, 8)`. Drop-on-overrun semantics in [Device.deliver]
	// rely on this being non-zero.
	streamChanDepth = 8

	// bulkInEndpoint is the RTL-SDR bulk-IN endpoint address. Hardware
	// invariant — every RTL2832U exposes the IQ stream here.
	bulkInEndpoint = 0x81
)

// StreamIQ resets the USB FIFO, kicks off a bulk-IN ring on the
// transport, and returns a buffered channel of complex64 samples.
// The channel closes when ctx cancels or [Device.Close] is called.
//
// Behaviour invariants — kept identical to rtlsdr_cgo.go:181-217 so
// the new driver is a drop-in replacement:
//
//   - Returns an error if a stream is already active on this Device.
//   - 8-deep buffered channel; senders use non-blocking select with
//     a default branch so slow consumers result in dropped buffers
//     rather than back-pressure on the kernel.
//   - The reaper goroutine inside [usb.Transport.StartBulkIn]
//     dispatches one onPacket call per completed URB.
//   - On ctx cancellation, [Device.cancelStream] runs StopBulkIn and
//     closes the channel.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.streamMu.Lock()
	defer d.streamMu.Unlock()
	if d.closed.Load() {
		return nil, ErrClosed
	}
	if d.out != nil {
		return nil, errors.New("rtlsdr-go: stream already active")
	}
	if err := d.demod.ResetBuffer(); err != nil {
		return nil, fmt.Errorf("rtlsdr-go: reset buffer: %w", err)
	}
	out := make(chan []complex64, streamChanDepth)
	d.out = out
	d.stopOnce = sync.Once{}

	if err := d.transport.StartBulkIn(bulkInEndpoint, asyncBufCount, asyncBufLen, d.deliver); err != nil {
		d.out = nil
		return nil, fmt.Errorf("rtlsdr-go: StartBulkIn: %w", err)
	}

	// Cancel goroutine: when ctx fires, tear the stream down. Mirrors
	// rtlsdr_cgo.go:211-214.
	go func() {
		<-ctx.Done()
		d.cancelStream()
	}()

	return out, nil
}

// deliver receives one bulk-IN buffer from the transport reaper,
// converts the u8 IQ pairs into complex64, and pushes the result
// onto the consumer channel. Drop-on-overrun if the consumer is
// behind. Bit-identical conversion math to rtlsdr_cgo.go:225-240
// so downstream DSP sees the same bit pattern from either backend.
func (d *Device) deliver(buf []byte) {
	samples := convertU8IQ(buf)
	d.streamMu.Lock()
	out := d.out
	d.streamMu.Unlock()
	if out == nil {
		return
	}
	select {
	case out <- samples:
	default:
		// Consumer can't keep up. Drop. The application's
		// `iq-underrun` Prometheus counter is the operator-facing
		// telemetry for this — see internal/metrics.
	}
}

// convertU8IQ translates a slice of unsigned-8-bit IQ pairs into
// normalized complex64 samples. The conversion is the bit-identical
// port of rtlsdr_cgo.go:225-240:
//
//   - DC bias of 127.5 (mid-range of u8) is subtracted.
//   - Result is divided by 127.5 to scale to [-1, +1).
//
// Pinned by [TestConvertU8IQ_BitIdenticalWithCGO] so any drift from
// the CGO driver shows up immediately.
func convertU8IQ(buf []byte) []complex64 {
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		i8 := float32(buf[2*i]) - 127.5
		q8 := float32(buf[2*i+1]) - 127.5
		out[i] = complex(i8/127.5, q8/127.5)
	}
	return out
}
