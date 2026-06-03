package voice

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// buildStream assembles a dibit stream of n voice superframes preceded
// by lead dibits of filler. Frame f of the stream carries mkFrame(f),
// so the caller can verify ordering across the whole stream. Burst A
// of each superframe is framed by the BS-Voice sync; bursts B–F use a
// non-voice sync (BS-Data) standing in for embedded signalling so the
// voice-only SyncDetector ignores them.
func buildStream(t *testing.T, n, lead int) (stream []uint8, frames [][]byte) {
	t.Helper()
	stream = make([]uint8, lead)
	total := n * FramesPerSuperframe
	for f := 0; f < total; f++ {
		frames = append(frames, mkFrame(f))
	}
	for s := 0; s < n; s++ {
		for b := 0; b < BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*FramesPerSuperframe + b*FramesPerBurst
			stream = append(stream, makeVoiceBurst(frames[base:base+FramesPerBurst], sync)...)
		}
	}
	return stream, frames
}

// buildInterleavedStream assembles a real 2-slot TDMA dibit stream: for
// each of n superframes the six bursts of slot A and slot B are woven
// burst-by-burst (A.b0, B.b0, A.b1, B.b1, … A.b5, B.b5), so each slot's
// own bursts are 264 dibits (two physical bursts) apart. Burst A of each
// slot's superframe carries BS-Voice sync; the rest carry BS-Data
// (standing in for embedded signalling). Slot B's frames seed from a
// large offset so the two slots' AMBE payloads are unmistakably
// distinct. lead prepends filler dibits to exercise non-aligned starts.
func buildInterleavedStream(t *testing.T, n, lead int) (stream []uint8, aFrames, bFrames [][]byte) {
	t.Helper()
	const bSeedOffset = 10000
	stream = make([]uint8, lead)
	total := n * FramesPerSuperframe
	for f := 0; f < total; f++ {
		aFrames = append(aFrames, mkFrame(f))
		bFrames = append(bFrames, mkFrame(f+bSeedOffset))
	}
	for s := 0; s < n; s++ {
		for b := 0; b < BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*FramesPerSuperframe + b*FramesPerBurst
			stream = append(stream, makeVoiceBurst(aFrames[base:base+FramesPerBurst], sync)...)
			stream = append(stream, makeVoiceBurst(bFrames[base:base+FramesPerBurst], sync)...)
		}
	}
	return stream, aFrames, bFrames
}

func checkFrames(t *testing.T, sf VoiceSuperframe, frames [][]byte, firstFrame int) {
	t.Helper()
	for i := 0; i < FramesPerSuperframe; i++ {
		if !bytes.Equal(sf.Frames[i], frames[firstFrame+i]) {
			t.Errorf("frame %d:\n got %v\nwant %v", i, sf.Frames[i], frames[firstFrame+i])
		}
	}
}

func TestDecoderExtractsSuperframe(t *testing.T) {
	stream, frames := buildStream(t, 1, 0)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if got[0].SyncName != "BS-Voice" {
		t.Errorf("SyncName = %q, want BS-Voice", got[0].SyncName)
	}
	if got[0].StartDibit != 0 {
		t.Errorf("StartDibit = %d, want 0", got[0].StartDibit)
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderTwoSuperframes(t *testing.T) {
	stream, frames := buildStream(t, 2, 0)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2", len(got))
	}
	checkFrames(t, got[0], frames, 0)
	checkFrames(t, got[1], frames, FramesPerSuperframe)
	if got[1].StartDibit != superframeDibits {
		t.Errorf("second StartDibit = %d, want %d", got[1].StartDibit, superframeDibits)
	}
}

func TestDecoderLeadingFiller(t *testing.T) {
	const lead = 137
	stream, frames := buildStream(t, 1, lead)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if got[0].StartDibit != lead {
		t.Errorf("StartDibit = %d, want %d", got[0].StartDibit, lead)
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderChunkedInput(t *testing.T) {
	stream, frames := buildStream(t, 1, 0)

	d := NewDecoder()
	var got []VoiceSuperframe
	const chunk = 100
	for off := 0; off < len(stream); off += chunk {
		end := off + chunk
		if end > len(stream) {
			end = len(stream)
		}
		got = append(got, d.Process(stream[off:end], off)...)
	}

	if len(got) != 1 {
		t.Fatalf("got %d superframes across chunks, want 1", len(got))
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderIncompleteSuperframe(t *testing.T) {
	stream, _ := buildStream(t, 1, 0)
	// Truncate to burst A plus two bursts — not a full superframe.
	truncated := stream[:3*dmr.BurstDibits]

	d := NewDecoder()
	if got := d.Process(truncated, 0); len(got) != 0 {
		t.Fatalf("got %d superframes from a partial superframe, want 0", len(got))
	}
}

func TestDecoderResetClearsState(t *testing.T) {
	stream, _ := buildStream(t, 1, 0)

	d := NewDecoder()
	// Feed burst A only, then reset before the rest arrives.
	d.Process(stream[:dmr.BurstDibits], 0)
	d.Reset()

	// A fresh full stream after reset must still decode cleanly.
	fresh, frames := buildStream(t, 1, 0)
	got := d.Process(fresh, 0)
	if len(got) != 1 {
		t.Fatalf("got %d superframes after reset, want 1", len(got))
	}
	checkFrames(t, got[0], frames, 0)
}

// TestInterleavedDecoderSeparatesSlots is the core 2-slot guard: fed a
// carrier whose two timeslots interleave burst-by-burst, the interleaved
// Decoder must emit one clean superframe per slot — each slice gathering
// only its own slot's bursts (264-dibit stride) and skipping the other
// slot's interleaved burst — tagged with distinct phases.
func TestInterleavedDecoderSeparatesSlots(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)

	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2 (one per timeslot)", len(got))
	}
	// Stream order: slot A's burst A (StartDibit 0) is detected before
	// slot B's (StartDibit 132, one physical burst later).
	a, b := got[0], got[1]
	if a.StartDibit != 0 || b.StartDibit != dmr.BurstDibits {
		t.Fatalf("StartDibits = %d, %d; want 0, %d", a.StartDibit, b.StartDibit, dmr.BurstDibits)
	}
	if a.Phase == b.Phase {
		t.Errorf("both slots reported Phase %d; want distinct phases", a.Phase)
	}
	// Each superframe must contain ONLY its own slot's AMBE frames — the
	// whole point of the stride. A single-slot (stride 1) decoder would
	// instead interleave A and B frames here and fail this check.
	checkFrames(t, a, aFrames, 0)
	checkFrames(t, b, bFrames, 0)
}

// TestLegacyDecoderMixesInterleavedSlots documents the bug the
// interleaved decoder fixes: a stride-1 Decoder run over a 2-slot stream
// locks burst A correctly but then grabs the WRONG (other-slot) bursts
// for B–F, so its output does not match either pure slot.
func TestLegacyDecoderMixesInterleavedSlots(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewDecoder() // stride 1 — wrong cadence for an interleaved carrier
	got := d.Process(stream, 0)
	if len(got) == 0 {
		t.Fatal("expected at least one (mis-assembled) superframe")
	}
	// got[0] anchors on slot A's burst A but slices contiguous bursts,
	// so its frames are an A/B interleave — equal to neither pure slot.
	mixed := got[0]
	aClean, bClean := true, true
	for i := 0; i < FramesPerSuperframe; i++ {
		if !bytes.Equal(mixed.Frames[i], aFrames[i]) {
			aClean = false
		}
		if !bytes.Equal(mixed.Frames[i], bFrames[i]) {
			bClean = false
		}
	}
	if aClean || bClean {
		t.Errorf("stride-1 decode unexpectedly matched a pure slot (aClean=%v bClean=%v); the interleaving guard is not exercised", aClean, bClean)
	}
}

// TestInterleavedDecoderPhaseStable confirms that across several
// superframes — and from a non-burst-aligned start offset — every
// superframe of a given slot carries the same Phase, so a downstream
// consumer can group a call's superframes by Phase for the call's
// lifetime.
func TestInterleavedDecoderPhaseStable(t *testing.T) {
	const n, lead = 3, 17
	stream, aFrames, bFrames := buildInterleavedStream(t, n, lead)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)
	if len(got) != 2*n {
		t.Fatalf("got %d superframes, want %d (%d per slot)", len(got), 2*n, n)
	}

	// Partition by phase; every superframe sharing slot A's phase must
	// carry slot A's frames, and likewise for B.
	aPhase := got[0].Phase
	var ai, bi int
	for _, sf := range got {
		if sf.Phase == aPhase {
			checkFrames(t, sf, aFrames, ai*FramesPerSuperframe)
			ai++
		} else {
			checkFrames(t, sf, bFrames, bi*FramesPerSuperframe)
			bi++
		}
	}
	if ai != n || bi != n {
		t.Errorf("phase partition = %d/%d superframes, want %d/%d", ai, bi, n, n)
	}
}

// TestInterleavedDecoderChunkedInput feeds the 2-slot stream in small
// chunks to exercise the cross-Process buffering of two simultaneously
// in-flight superframe anchors.
func TestInterleavedDecoderChunkedInput(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewInterleavedDecoder()
	var got []VoiceSuperframe
	const chunk = 137
	for off := 0; off < len(stream); off += chunk {
		end := off + chunk
		if end > len(stream) {
			end = len(stream)
		}
		got = append(got, d.Process(stream[off:end], off)...)
	}
	if len(got) != 2 {
		t.Fatalf("got %d superframes across chunks, want 2", len(got))
	}
	checkFrames(t, got[0], aFrames, 0)
	checkFrames(t, got[1], bFrames, 0)
}
