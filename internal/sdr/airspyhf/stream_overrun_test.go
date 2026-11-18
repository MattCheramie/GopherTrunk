package airspyhf

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// streamScript is the control traffic StreamIQ issues: receiver on at start,
// off on teardown.
func streamScript() []usb.CtrlExchange {
	return []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}
}

// countDrops installs an IQ-drop observer for the test's duration.
func countDrops(t *testing.T) *atomic.Uint64 {
	t.Helper()
	var drops atomic.Uint64
	sdr.SetIQDropObserver(func(sdr.Info) { drops.Add(1) })
	t.Cleanup(func() { sdr.SetIQDropObserver(nil) })
	return &drops
}

// TestStreamIQDropsInsteadOfBlockingWhenConsumerStalls is the #1375
// regression: a consumer that stops reading must not stop the device being
// read. The transport posts its next read only after onPacket returns, so a
// blocking send there wedges the USB reaper (on macOS every reaper blocks at
// once and the stream stops with no error, the Airspy freeze), and the IQ
// lost meanwhile is never counted. The mock delivers every packet from one
// goroutine; with a blocking onPacket it stops after the channel fills and no
// drop is ever reported.
func TestStreamIQDropsInsteadOfBlockingWhenConsumerStalls(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = streamScript()
	const packets = 64
	mt.BulkPackets = make([][]byte, packets)
	for i := range mt.BulkPackets {
		mt.BulkPackets[i] = make([]byte, 64) // 16 IQ samples each
	}
	drops := countDrops(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// Never read ch: a stalled downstream consumer.
	want := uint64(packets - cap(ch))
	deadline := time.Now().Add(2 * time.Second)
	for drops.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("got %d drops, want %d: a blocking onPacket stopped the USB reads", drops.Load(), want)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

// TestStreamIQAbsorbsConsumerPauseAtStreamRate: the stream channel holds a
// consumer pause measured in time at the stream's rate, not a fixed chunk
// count. A 16 KiB URB is 4096 IQ samples (int16 I and Q), so the old 8-deep
// channel held 43 ms at 768 kS/s. Here 40 full-size URBs (213 ms at 768 kS/s)
// arrive while the consumer is paused; none may be lost.
func TestStreamIQAbsorbsConsumerPauseAtStreamRate(t *testing.T) {
	dev, mt := withDevice(t)
	dev.rateHz = 768_000
	mt.Script = streamScript()
	const packets = 40
	mt.BulkPackets = make([][]byte, packets)
	for i := range mt.BulkPackets {
		mt.BulkPackets[i] = make([]byte, usb.DefaultBufferLen)
	}
	drops := countDrops(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(ch)+int(drops.Load()) < packets {
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d chunks buffered and %d dropped: the stream channel (cap %d) is sized by chunk count, not time",
				len(ch), packets, drops.Load(), cap(ch))
		}
		time.Sleep(2 * time.Millisecond)
	}
	if n := drops.Load(); n != 0 {
		t.Fatalf("dropped %d of %d chunks in a 213 ms consumer pause at 768 kS/s (cap %d)", n, packets, cap(ch))
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestStreamDepthForSizesByTime(t *testing.T) {
	cases := []struct {
		rate uint32
		want int
	}{
		{0, minStreamChanDepth},      // rate unknown (no firmware table yet)
		{912_000, 90},                // ceil(0.4 · 912e3 / 4096)
		{768_000, 75},                // ceil(0.4 · 768e3 / 4096)
		{192_000, 19},                // ceil(0.4 · 192e3 / 4096)
		{50_000, minStreamChanDepth}, // 4.9 chunks: the floor
	}
	for _, c := range cases {
		if got := streamDepthFor(c.rate, usb.DefaultBufferLen/4); got != c.want {
			t.Errorf("streamDepthFor(%d) = %d, want %d", c.rate, got, c.want)
		}
	}
}
