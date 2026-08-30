package dpmr

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildTCHFieldDibits encodes four 49-bit payloads into the 144-dibit
// (288-bit) TCH field of one dPMR voice frame.
func buildTCHFieldDibits(t *testing.T, payloads [4][]byte) []uint8 {
	t.Helper()
	bits := make([]byte, 0, TCHFramesPerBurst*dpmrAMBEOnAirBits)
	for i, p := range payloads {
		frame, err := EncodeTCHFrame(p)
		if err != nil {
			t.Fatalf("EncodeTCHFrame(%d): %v", i, err)
		}
		bits = append(bits, frame...)
	}
	dibits := framing.BitsToDibits(bits)
	if len(dibits) != TCHFieldDibits {
		t.Fatalf("tch field = %d dibits, want %d", len(dibits), TCHFieldDibits)
	}
	return dibits
}

// buildVoiceFrame assembles a full dPMR voice frame preceded by warm-up
// filler: warmup(SyncDibits) + sync(24) + CCH(24, zeros) + TCH(144).
// The warm-up primes the SyncDetector (it needs SyncDibits of history
// before it can match), mirroring a live stream where the sync is never
// the very first dibit. sync selects FS1 (superframe start) or FS2
// (mid-superframe) so the same builder exercises both anchors.
func buildVoiceFrame(t *testing.T, sync []uint8, tch []uint8) []uint8 {
	t.Helper()
	frame := make([]uint8, 0, SyncDibits+SyncDibits+postSyncDibitsTraffic)
	frame = append(frame, make([]uint8, SyncDibits)...) // warm-up filler
	frame = append(frame, sync...)
	frame = append(frame, make([]uint8, CCHDibits)...) // CCH (skipped)
	frame = append(frame, tch...)
	return frame
}

func TestTrafficChannelDecodesFS1Frame(t *testing.T) {
	payloads := [4][]byte{mkTCHInfo(11), mkTCHInfo(22), mkTCHInfo(33), mkTCHInfo(44)}
	frame := buildVoiceFrame(t, FS1Dibits(), buildTCHFieldDibits(t, payloads))

	var got [][]TCHFrame
	tc := NewTrafficChannel(func(frames []TCHFrame) {
		got = append(got, frames)
	})
	tc.Process(frame, 0)

	if len(got) != 1 {
		t.Fatalf("sink called %d times, want 1", len(got))
	}
	if len(got[0]) != TCHFramesPerBurst {
		t.Fatalf("got %d voice frames, want %d", len(got[0]), TCHFramesPerBurst)
	}
	for i, f := range got[0] {
		for j := range payloads[i] {
			if f.Payload[j] != payloads[i][j] {
				t.Fatalf("voice frame %d bit %d: got %d want %d", i, j, f.Payload[j], payloads[i][j])
			}
		}
	}
}

func TestTrafficChannelDecodesFS2Frame(t *testing.T) {
	payloads := [4][]byte{mkTCHInfo(1), mkTCHInfo(2), mkTCHInfo(3), mkTCHInfo(4)}
	frame := buildVoiceFrame(t, FS2Dibits(), buildTCHFieldDibits(t, payloads))

	var got [][]TCHFrame
	tc := NewTrafficChannel(func(frames []TCHFrame) { got = append(got, frames) })
	tc.Process(frame, 0)

	if len(got) != 1 || len(got[0]) != TCHFramesPerBurst {
		t.Fatalf("FS2-anchored frame not decoded: %d batches", len(got))
	}
}

// TestTrafficChannelIgnoresCSBKBurst confirms an FS3 (signalling) burst
// never fires the voice sink: FS3 differs from FS1/FS2 by far more than
// the detector tolerance, so a control channel's CSBK traffic cannot be
// misread as voice.
func TestTrafficChannelIgnoresCSBKBurst(t *testing.T) {
	burst := make([]uint8, 0, SyncDibits+SyncDibits+40)
	burst = append(burst, make([]uint8, SyncDibits)...) // warm-up
	burst = append(burst, FS3Dibits()...)
	burst = append(burst, make([]uint8, 40)...) // CSBK dibits

	called := false
	tc := NewTrafficChannel(func(frames []TCHFrame) { called = true })
	tc.Process(burst, 0)

	if called {
		t.Error("sink should not fire for an FS3/CSBK burst")
	}
}

func TestTrafficChannelSurvivesChunkedFeed(t *testing.T) {
	payloads := [4][]byte{mkTCHInfo(5), mkTCHInfo(6), mkTCHInfo(7), mkTCHInfo(8)}
	frame := buildVoiceFrame(t, FS1Dibits(), buildTCHFieldDibits(t, payloads))

	var got [][]TCHFrame
	tc := NewTrafficChannel(func(frames []TCHFrame) { got = append(got, frames) })
	// Feed one dibit at a time to exercise the cross-call countdown.
	base := 0
	for i := 0; i < len(frame); i++ {
		tc.Process(frame[i:i+1], base)
		base++
	}
	if len(got) != 1 || len(got[0]) != TCHFramesPerBurst {
		t.Fatalf("chunked feed decoded %d frame-batches", len(got))
	}
}
