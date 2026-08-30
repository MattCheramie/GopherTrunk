package dstar

import (
	"testing"
)

// buildDVFrame assembles one 96-bit DV frame: 72 encoded voice bits +
// a 24-bit data field (the Slow Data sync when withSync, zeros
// otherwise — zeros are >1 bit away from the sync pattern so they
// never false-match at tolerance 1).
func buildDVFrame(t *testing.T, payload []byte, withSync bool) []uint8 {
	t.Helper()
	voice, err := EncodeDVVoiceBits(payload)
	if err != nil {
		t.Fatalf("EncodeDVVoiceBits: %v", err)
	}
	frame := make([]uint8, 0, DVFrameBits)
	frame = append(frame, voice...)
	if withSync {
		frame = append(frame, SlowDataSyncBitsSlice()...)
	} else {
		frame = append(frame, make([]uint8, DVDataBits)...)
	}
	return frame
}

// buildDVStream lays payloads onto the DV cadence: frame 0 and every
// DVFramesPerSyncCycle-th frame carry the Slow Data sync; the stream is
// preceded by a zero warm-up so the detector history and the rolling
// 96-bit voice window are both primed before the first sync.
func buildDVStream(t *testing.T, payloads [][]byte) []uint8 {
	t.Helper()
	stream := make([]uint8, 0, DVFrameBits*(len(payloads)+1))
	stream = append(stream, make([]uint8, DVFrameBits)...) // warm-up
	for i, p := range payloads {
		stream = append(stream, buildDVFrame(t, p, i%DVFramesPerSyncCycle == 0)...)
	}
	return stream
}

func checkPayloads(t *testing.T, got []DVFrame, want [][]byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("decoded %d frames, want %d", len(got), len(want))
	}
	for i := range want {
		for j := range want[i] {
			if got[i].Payload[j] != want[i][j] {
				t.Fatalf("frame %d bit %d: got %d want %d", i, j, got[i].Payload[j], want[i][j])
			}
		}
	}
}

// TestVoiceChannelDecodesAnchoredCadence spans a full sync cycle plus
// the next sync frame — 22 frames — asserting every frame decodes
// exactly once and in order. Frame 21 is both the free-run cadence's
// 21st frame AND a sync frame, so this also pins the no-duplicate
// guard at the frame/sync boundary.
func TestVoiceChannelDecodesAnchoredCadence(t *testing.T) {
	var payloads [][]byte
	for i := 0; i < DVFramesPerSyncCycle+1; i++ {
		payloads = append(payloads, mkDVInfo(uint32(i)))
	}
	stream := buildDVStream(t, payloads)

	var got []DVFrame
	vc := NewVoiceChannel(func(f DVFrame) { got = append(got, f) })
	vc.Process(stream, 0)

	checkPayloads(t, got, payloads)
}

// TestVoiceChannelDropsAnchorWithoutSync confirms the free-run stops
// after two sync-less cycles: one sync frame followed by 2×21 sync-less
// frames decodes 1+42 frames, and further sync-less bits decode
// nothing until a new sync arrives.
func TestVoiceChannelDropsAnchorWithoutSync(t *testing.T) {
	sync := mkDVInfo(100)
	var free [][]byte
	for i := 0; i < 2*DVFramesPerSyncCycle; i++ {
		free = append(free, mkDVInfo(uint32(200+i)))
	}

	stream := make([]uint8, 0, DVFrameBits*50)
	stream = append(stream, make([]uint8, DVFrameBits)...) // warm-up
	stream = append(stream, buildDVFrame(t, sync, true)...)
	for _, p := range free {
		stream = append(stream, buildDVFrame(t, p, false)...)
	}
	// Post-drop garbage: more sync-less frames that must NOT decode.
	for i := 0; i < 3; i++ {
		stream = append(stream, buildDVFrame(t, mkDVInfo(uint32(900+i)), false)...)
	}

	var got []DVFrame
	vc := NewVoiceChannel(func(f DVFrame) { got = append(got, f) })
	vc.Process(stream, 0)

	want := 1 + 2*DVFramesPerSyncCycle
	if len(got) != want {
		t.Fatalf("decoded %d frames, want %d (anchor should drop after 2 sync-less cycles)", len(got), want)
	}

	// A fresh sync re-anchors and decodes again.
	before := len(got)
	vc.Process(buildDVFrame(t, mkDVInfo(999), true), len(stream))
	if len(got) != before+1 {
		t.Fatalf("re-anchor sync decoded %d new frames, want 1", len(got)-before)
	}
}

// TestVoiceChannelSurvivesChunkedFeed feeds the cadence one bit at a
// time to exercise the cross-call countdown + rolling window.
func TestVoiceChannelSurvivesChunkedFeed(t *testing.T) {
	var payloads [][]byte
	for i := 0; i < 5; i++ {
		payloads = append(payloads, mkDVInfo(uint32(50+i)))
	}
	stream := buildDVStream(t, payloads)

	var got []DVFrame
	vc := NewVoiceChannel(func(f DVFrame) { got = append(got, f) })
	base := 0
	for i := 0; i < len(stream); i++ {
		vc.Process(stream[i:i+1], base)
		base++
	}
	checkPayloads(t, got, payloads)
}

// TestVoiceChannelResyncsAfterBitSlip inserts a stray bit mid-stream
// (a symbol slip) and confirms the next Slow Data sync re-anchors the
// cadence: the frames of the second cycle decode correctly even though
// the free-run ran misaligned in between.
func TestVoiceChannelResyncsAfterBitSlip(t *testing.T) {
	var cycle0, cycle1 [][]byte
	for i := 0; i < DVFramesPerSyncCycle; i++ {
		cycle0 = append(cycle0, mkDVInfo(uint32(300+i)))
		cycle1 = append(cycle1, mkDVInfo(uint32(400+i)))
	}

	stream := make([]uint8, 0, DVFrameBits*50)
	stream = append(stream, make([]uint8, DVFrameBits)...) // warm-up
	for i, p := range cycle0 {
		stream = append(stream, buildDVFrame(t, p, i == 0)...)
	}
	stream = append(stream, 1) // the slip
	for i, p := range cycle1 {
		stream = append(stream, buildDVFrame(t, p, i == 0)...)
	}

	var got []DVFrame
	vc := NewVoiceChannel(func(f DVFrame) { got = append(got, f) })
	vc.Process(stream, 0)

	// Cycle 1 must appear intact at the tail of the decode sequence.
	if len(got) < DVFramesPerSyncCycle {
		t.Fatalf("decoded only %d frames", len(got))
	}
	tail := got[len(got)-DVFramesPerSyncCycle:]
	checkPayloads(t, tail, cycle1)
}
