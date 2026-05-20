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
