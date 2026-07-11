package airspy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestStreamIQDropsInsteadOfBlockingWhenConsumerStalls is the regression for the
// Airspy decode freeze: a stalled IQ consumer must never wedge the USB reaper.
//
// Each usb transport bulkLoop goroutine posts its next ReadPipe only after
// onPacket returns, so a blocking send anywhere in the deliver path stops the
// device from being read. On macOS DefaultRingBuffers=32 reapers all block at
// once, no USB reads stay outstanding, the endpoint FIFO overflows and the whole
// stream silently wedges with no error/EOF/log. The deliver path is now two
// stages — the reaper copies and hands off to rawCh, and a single converter
// goroutine runs the FIR and sends to `out` — and BOTH sends must shed on
// overflow (counted via sdr.NotifyIQDrop) rather than block, so the reaper keeps
// cycling and the device stays alive.
//
// Here the consumer never drains the returned channel. If either stage blocked,
// the mock's single delivery goroutine would wedge once `out` (8-deep) fills and
// no further drops would be observed → this test times out below its target. With
// non-blocking sheds at both stages, every packet past the buffers is dropped and
// counted. By conservation exactly packets-cap(out) chunks are shed regardless of
// rawCh depth (out fills to 8 and stays there; the converter drains rawCh empty).
func TestStreamIQDropsInsteadOfBlockingWhenConsumerStalls(t *testing.T) {
	dev, mt := withDevice(t)
	// StreamIQ flips the receiver on at start and off again on teardown.
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}

	// Feed far more packets than the 8-deep primary channel can hold, back to
	// back, to a consumer that never reads. 64 bytes → 16 complex samples each.
	const packets = 64
	const bufBytes = 64
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
	// Deliberately never read ch — simulate a stalled downstream consumer.

	// The channel buffers 8, so exactly packets-8 chunks must be shed. Poll with
	// a deadline: a blocking reaper never reaches this count and the test fails
	// loudly instead of hanging forever.
	wantMin := uint64(packets - cap(ch))
	deadline := time.Now().Add(2 * time.Second)
	for drops.Load() < wantMin {
		if time.Now().After(deadline) {
			t.Fatalf("reaper wedged: got %d drops, want >= %d — a blocking onPacket "+
				"stopped the USB reads (freeze regression)", drops.Load(), wantMin)
		}
		time.Sleep(2 * time.Millisecond)
	}

	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}
