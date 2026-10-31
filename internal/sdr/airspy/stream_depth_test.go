package airspy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestStreamIQAbsorbsConsumerPauseAtStreamRate is the #1321 regression: the
// output channel must hold a consumer pause measured in TIME at the stream's
// rate, not a fixed chunk count. A real URB is usb.DefaultBufferLen (16 KiB) =
// 4096 IQ samples, so the old 8-deep channel held ~11 ms at 3 MS/s: the
// reporter's Pi 4 shed ~9 chunks/min with 4 TETRA cells and its cores mostly
// idle. Here 64 full-size URBs (~87 ms of IQ at 3 MS/s) arrive while the
// consumer is paused; none may be dropped. The old code drops 56.
func TestStreamIQAbsorbsConsumerPauseAtStreamRate(t *testing.T) {
	dev, mt := withDevice(t)
	dev.lastReqRate = 3_000_000 // the Mini's 3 MS/s; no rate table ⇒ used as-is
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}
	const packets = 64
	mt.BulkPackets = make([][]byte, packets)
	for i := range mt.BulkPackets {
		mt.BulkPackets[i] = make([]byte, usb.DefaultBufferLen)
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
	// The consumer is "paused": nothing reads ch until every URB is in.
	deadline := time.Now().Add(5 * time.Second)
	for len(ch)+int(drops.Load()) < packets {
		if time.Now().After(deadline) {
			t.Fatalf("only %d chunks buffered, %d dropped of %d", len(ch), drops.Load(), packets)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if n := drops.Load(); n != 0 {
		t.Fatalf("dropped %d of %d chunks during an ~87 ms consumer pause at 3 MS/s "+
			"(channel cap %d): the stream buffer is sized by chunk count, not time", n, packets, cap(ch))
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestStreamDepthForSizesByTime(t *testing.T) {
	cases := []struct {
		rate uint32
		per  int
		want int
	}{
		{0, 4096, minStreamChanDepth},      // rate unknown
		{3_000_000, 4096, 293},             // Mini 3 MS/s: 1.2 M samples / 4096
		{10_000_000, 4096, 977},            // R2 10 MS/s
		{6_000_000, 8192, 293},             // 8-bit HydraSDR payload
		{20_000, 4096, minStreamChanDepth}, // floor
	}
	for _, c := range cases {
		if got := streamDepthFor(c.rate, c.per); got != c.want {
			t.Errorf("streamDepthFor(%d, %d) = %d, want %d", c.rate, c.per, got, c.want)
		}
	}
}
