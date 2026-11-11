package wmbus

import (
	"path/filepath"
	"testing"
)

// TestReceiverKeepsChipClockThroughLongRunsOffCarrier replays a real-air
// C mode telegram from the #1256 reporter's 8 Nov walk (QDS 95951473, a
// heat cost allocator at -16.8 dBFS, clean flat burst) through a receiver
// whose carrier estimate sits where the live run's did: 3.4 kHz off this
// meter, learned from a stronger meter a few seconds earlier. Live, the
// telegram failed CRC blocks 4 and 5 of 5; replayed cold on its own it
// decoded, so the failure rode in receiver state.
//
// The chip clock was the state. The telegram's tail carries runs of 107
// and 74 identical chips (NRZ zero bytes), over which the clock free-runs
// on its period estimate, so it needs that estimate accurate to better
// than half a chip over 107 chips. Right before the first run a noise
// spike crossed the threshold twice within a third of a chip, and the
// loop took the second crossing as a chip edge half a chip out of place:
// a 0.2 % kick to the period and a third of a sample to the phase, enough
// that the clock drifted through the runs (4.018 -> 4.047 samples per
// chip, measured, against a true 4.00) and the last two CRC blocks
// failed. The clock now ignores a crossing within half a chip of the
// previous one, which no chip edge can be. Replayed cold the same
// telegram decoded because the spike met different loop state.
func TestReceiverKeepsChipClockThroughLongRunsOffCarrier(t *testing.T) {
	iq, rate := loadIQWav(t, filepath.Join("testdata", "issue1256_qds_95951473_longrun_offcarrier.wav"))
	var got Frame
	var frames int
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, _ FrameInfo) {
		frames++
		got = f
	}})
	if err != nil {
		t.Fatal(err)
	}
	// What the live run had learned from the meters before this one
	// (the log's carrier -57.0 kHz; this telegram's own sits at -53.6).
	rx.afc(-57_000)
	for j := 0; j < len(iq); j += 8192 {
		rx.Process(iq[j:min(j+8192, len(iq))])
	}
	rx.Flush()
	if frames != 1 {
		t.Fatalf("framed %d telegrams, want 1", frames)
	}
	if !got.CRCOK {
		t.Fatalf("the telegram failed its CRC (raw %d bytes); the chip clock lost the tail of the frame", len(got.Raw))
	}
	if len(got.Raw) != 84 {
		t.Errorf("raw length %d, want 84 (74 data + 5 block CRCs)", len(got.Raw))
	}
}
