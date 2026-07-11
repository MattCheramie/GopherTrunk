package airspy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestStreamIQReaperNotSerializedByConversion is the regression for the macOS
// 10 MS/s abort (usb: ReadPipe: 0xe00002eb / kIOReturnAborted). On macOS the USB
// transport runs 32 concurrent bulk-IN reapers, each of which re-posts its next
// ReadPipe only after onPacket returns. When onPacket ran the heavy real→IQ FIR
// conversion inline (serialised on the converter mutex), the reapers stalled on
// conversion instead of re-posting reads; at ~40 MB/s the host controller's
// outstanding-transfer queue collapsed and macOS aborted the pipe. onPacket must
// therefore NOT run the conversion — it must only copy the payload and hand it to
// the single converter goroutine, returning immediately.
//
// The mock delivers packets from one goroutine, so it can't reproduce the
// 32-thread collapse directly. Instead this pins the property that fixes it:
// onPacket returns without running the conversion.
//
// The discriminator is a hard physical floor, not a fragile rate estimate. The
// old inline-conversion code ran processRaw (cost tConv per buffer) on the single
// deliver goroutine before it could even attempt the (dropping) send, so it
// cannot shed wantMin chunks in less than ~wantMin×tConv — sequential conversion
// is unavoidable. We set the deadline safely below that floor (0.8×wantMin×tConv):
// the old code physically cannot reach wantMin drops in time and the test fails;
// the decoupled reaper does no conversion in onPacket and blows past wantMin with
// wide margin. tConv is calibrated in-process, so the bound tracks -race and slow
// CI proportionally.
func TestStreamIQReaperNotSerializedByConversion(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}

	// Calibrate the cost of one full-size FIR conversion (the work onPacket used
	// to do inline). A 16 KiB URB is the real production payload size.
	const bufBytes = usb.DefaultBufferLen
	calBuf := make([]byte, bufBytes)
	for i := range calBuf {
		calBuf[i] = byte(i)
	}
	const calIters = 16
	calConv := newIQConverter()
	calStart := time.Now()
	for i := 0; i < calIters; i++ {
		calConv.processRaw(calBuf)
	}
	tConv := time.Since(calStart) / calIters
	if tConv <= 0 {
		tConv = time.Microsecond // implausible, but never divide the deadline by zero
	}

	// Feed far more full-size packets than the pipeline can buffer, back to back,
	// to a consumer that never drains — so every packet past the buffers is shed.
	const (
		packets = 256
		wantMin = 150
	)
	mt.BulkPackets = make([][]byte, packets)
	for i := range mt.BulkPackets {
		mt.BulkPackets[i] = make([]byte, bufBytes)
	}

	var drops atomic.Uint64
	sdr.SetIQDropObserver(func(sdr.Info) { drops.Add(1) })
	t.Cleanup(func() { sdr.SetIQDropObserver(nil) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	_ = ch // Deliberately never read ch — a stalled downstream consumer.

	// 0.8×wantMin×tConv: below the ~wantMin×tConv floor the old inline-conversion
	// code cannot beat (it converts each packet sequentially before dropping),
	// well above what the decoupled reaper needs.
	deadline := time.Duration(wantMin) * tConv * 4 / 5
	start := time.Now()
	for drops.Load() < wantMin {
		if elapsed := time.Since(start); elapsed > deadline {
			t.Fatalf("reaper serialized behind conversion (decoupling regressed): "+
				"got %d drops in %v, want >= %d within %v (tConv=%v) — onPacket must "+
				"copy and hand off, not run the FIR inline",
				drops.Load(), elapsed, wantMin, deadline, tConv)
		}
		time.Sleep(tConv)
	}

	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}
