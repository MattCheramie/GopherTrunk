package ccdecoder

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// A same-carrier voice stream reports what it lost while it is still open:
// the chunks the fan-out dropped to it, and the chunks the control SDR's
// driver dropped upstream since the stream opened (issue #1363). Before,
// the source had no loss reader at all, so the composer fell back to the
// driver count of the tap's own pseudo-serial, which is always zero.
func TestCCVoiceSourceReportsStreamLoss(t *testing.T) {
	const physical = "CTRL-SDR-1363"
	info := sdr.Info{Driver: "test", Serial: physical}
	sdr.NotifyIQDrop(info) // before the stream: must not count

	d := &Decoder{voiceFan: newVoiceFanout(nil, 2, nil), activeFreqHz: 467_913_000, pipelineRateHz: 144_000}
	src := NewCCVoiceSource(func() *Decoder { return d }, "cc:same-carrier:1")
	src.SetPhysicalSerial(func() string { return physical })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, lost, err := src.StreamIQWithLoss(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := lost(); got != 0 {
		t.Fatalf("loss at open = %d, want 0", got)
	}
	// Depth 2 and nobody reading: the third and fourth chunks are dropped.
	for i := 0; i < 4; i++ {
		d.voiceFan.broadcast(make([]complex64, 8))
	}
	if got := lost(); got != 2 {
		t.Fatalf("loss after 2 fan-out drops = %d, want 2", got)
	}
	sdr.NotifyIQDrop(info)
	sdr.NotifyIQDrop(info)
	sdr.NotifyIQDrop(info)
	if got := lost(); got != 5 {
		t.Fatalf("loss after 3 more driver drops = %d, want 5", got)
	}
	// Chunks the decoder shed at its own decode queue never reach the
	// fan-out either.
	d.decodeDrops.Add(2)
	if got := lost(); got != 7 {
		t.Fatalf("loss after 2 decode-queue drops = %d, want 7", got)
	}
}

// Without a known control SDR the reader still counts the fan-out's drops.
func TestCCVoiceSourceLossWithoutPhysicalSerial(t *testing.T) {
	d := &Decoder{voiceFan: newVoiceFanout(nil, 1, nil)}
	src := NewCCVoiceSource(func() *Decoder { return d }, "cc:same-carrier:1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, lost, err := src.StreamIQWithLoss(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d.voiceFan.broadcast(make([]complex64, 8))
	d.voiceFan.broadcast(make([]complex64, 8))
	if got := lost(); got != 1 {
		t.Fatalf("loss = %d, want 1", got)
	}
	if _, _, err := NewCCVoiceSource(func() *Decoder { return nil }, "x").StreamIQWithLoss(ctx); err == nil {
		t.Fatal("no decoder: want an error")
	}
}
