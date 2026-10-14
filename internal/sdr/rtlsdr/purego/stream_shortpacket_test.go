package purego

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// shortPacketDevice provisions a device the way StreamIQ does for depth
// queued chunks, without a transport.
func shortPacketDevice(depth int) *Device {
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), &fakeTuner{})
	d.ring = make([][]complex64, depth+2)
	for i := range d.ring {
		d.ring[i] = make([]complex64, asyncBufLen/2)
	}
	d.out = make(chan []complex64, depth)
	return d
}

// shortPacketStream is n packets of pktBytes each, every byte distinct
// enough (a running counter) that a reordered or lost sample shows.
func shortPacketStream(n, pktBytes int) [][]byte {
	pkts := make([][]byte, n)
	c := 0
	for i := range pkts {
		pkts[i] = make([]byte, pktBytes)
		for j := range pkts[i] {
			pkts[i][j] = byte(c)
			c = c*7 + 3
		}
	}
	return pkts
}

// TestDeliverShortPacketsSurviveConsumerStall is the issue #1256 regression.
// On the reporter's phone the kernel completes RTL-SDR bulk URBs short:
// their own numbers (639 dropped chunks while a 10.6 s, 1.8 MS/s capture
// ran only 5 % slow) put a dropped chunk near 1.7k samples, not the 8k of a
// full 16 KiB URB. A queue sized in chunks then holds a fifth of the time it
// was sized for. Here the consumer pauses for 100 ms at 1.6 MS/s while
// 3.2 KB packets arrive: 100 packets into a 32-deep queue. Before the fix
// each packet took a queue entry, so 68 of them were dropped; packed into
// full ring slots, the whole 100 ms fits in 20.
func TestDeliverShortPacketsSurviveConsumerStall(t *testing.T) {
	const (
		rate     = 1_600_000
		pktBytes = 3200 // 1600 samples, 1 ms at 1.6 MS/s
		stall    = rate / 10 * 2 / pktBytes
	)
	d := shortPacketDevice(streamChanDepth)
	pkts := shortPacketStream(stall, pktBytes)
	for _, p := range pkts {
		d.deliver(p)
	}
	if got := d.DroppedChunks(); got != 0 {
		t.Fatalf("dropped %d chunks during a 100 ms consumer stall of %d-byte packets, want 0", got, pktBytes)
	}

	// Every sample arrives once, in order: what was queued plus what is
	// still being packed into the current slot.
	var got []complex64
	for len(d.out) > 0 {
		got = append(got, <-d.out...)
	}
	got = append(got, d.ring[d.ringIdx][:d.ringFill]...)
	var all []byte
	for _, p := range pkts {
		all = append(all, p...)
	}
	want := convertU8IQ(all)
	if len(got) != len(want) {
		t.Fatalf("delivered %d samples, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d = %v, want %v (packing reordered or lost IQ)", i, got[i], want[i])
		}
	}
}

// TestDeliverShortPacketsFlowImmediatelyWhenCaughtUp: packing must not add
// latency while the consumer keeps up. With nothing queued, each packet
// goes out on its own, as before the fix.
func TestDeliverShortPacketsFlowImmediatelyWhenCaughtUp(t *testing.T) {
	d := shortPacketDevice(streamChanDepth)
	for i, p := range shortPacketStream(20, 3200) {
		d.deliver(p)
		select {
		case c := <-d.out:
			if len(c) != 1600 {
				t.Fatalf("packet %d: chunk of %d samples, want 1600", i, len(c))
			}
		default:
			t.Fatalf("packet %d was held back with an empty queue", i)
		}
	}
}

// TestDeliverPacksAcrossSlotBoundary: a packet that overflows the slot being
// packed fills it, queues it and carries the rest into the next slot.
func TestDeliverPacksAcrossSlotBoundary(t *testing.T) {
	d := shortPacketDevice(4)
	d.out <- nil                       // consumer behind: something is queued
	pkts := shortPacketStream(7, 5000) // 2500 samples each; slot holds 8192
	for _, p := range pkts {
		d.deliver(p)
	}
	<-d.out
	var lens []int
	for len(d.out) > 0 {
		lens = append(lens, len(<-d.out))
	}
	if len(lens) != 2 || lens[0] != asyncBufLen/2 || lens[1] != asyncBufLen/2 {
		t.Fatalf("queued chunk lengths %v, want two full slots of %d", lens, asyncBufLen/2)
	}
	if d.ringFill != 7*2500-2*asyncBufLen/2 {
		t.Errorf("packing slot holds %d samples, want %d", d.ringFill, 7*2500-2*asyncBufLen/2)
	}
}

// TestDeliverDropsWholeSlotsWhenQueueFull: a consumer that stays stalled
// still loses IQ, counted once per full slot of samples.
func TestDeliverDropsWholeSlotsWhenQueueFull(t *testing.T) {
	d := shortPacketDevice(2)
	for _, p := range shortPacketStream(20, asyncBufLen) {
		d.deliver(p)
	}
	if got := d.DroppedChunks(); got != 18 {
		t.Errorf("DroppedChunks = %d, want 18 (20 full packets into a 2-deep queue)", got)
	}
	if p, b := d.USBPacketStats(); p != 20 || b != 20*asyncBufLen {
		t.Errorf("USBPacketStats = %d packets / %d bytes, want 20 / %d", p, b, 20*asyncBufLen)
	}
}

// TestStreamDepthIsTimeSized pins the queue to streamBufferLatency of IQ at
// the stream rate, never below streamChanDepth.
func TestStreamDepthIsTimeSized(t *testing.T) {
	for _, tc := range []struct {
		rate uint32
		want int
	}{
		{0, streamChanDepth},
		{250_000, streamChanDepth},
		{1_600_000, 79},
		{2_400_000, 118},
		{3_200_000, 157},
	} {
		if got := streamDepthFor(tc.rate); got != tc.want {
			t.Errorf("streamDepthFor(%d) = %d, want %d", tc.rate, got, tc.want)
		}
	}
}
