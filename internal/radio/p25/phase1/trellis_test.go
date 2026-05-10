package phase1

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestTrellisRoundTrip(t *testing.T) {
	in := make([]uint8, 48)
	r := rand.New(rand.NewSource(1))
	for i := range in {
		in[i] = uint8(r.Intn(4))
	}
	channel := EncodeTrellis(in)
	if len(channel) != 98 {
		t.Fatalf("encoded len = %d, want 98", len(channel))
	}
	got, metric := DecodeTrellis(channel)
	if metric != 0 {
		t.Errorf("clean channel metric = %d, want 0", metric)
	}
	if !bytes.Equal(got, in) {
		t.Errorf("round-trip mismatch:\n got %v\nwant %v", got, in)
	}
}

func TestTrellisDecodesWithSingleErrors(t *testing.T) {
	in := make([]uint8, 48)
	r := rand.New(rand.NewSource(2))
	for i := range in {
		in[i] = uint8(r.Intn(4))
	}
	channel := EncodeTrellis(in)
	// Flip one bit in one channel dibit. The decoder should still
	// recover the original input (4-state, 1/2-rate trellis with d_free
	// large enough to handle isolated single-dibit errors).
	channel[33] ^= 0b01
	got, metric := DecodeTrellis(channel)
	if !bytes.Equal(got, in) {
		t.Errorf("single-error decode mismatch:\n got %v\nwant %v", got, in)
	}
	if metric == 0 {
		t.Errorf("metric = 0 after error, want > 0")
	}
}

func TestTrellisOutputDibitsInRange(t *testing.T) {
	in := make([]uint8, 48)
	out := EncodeTrellis(in)
	for i, d := range out {
		if d > 3 {
			t.Errorf("channel[%d] = %d, want 0..3", i, d)
		}
	}
}

func TestInterleaveRoundTrip(t *testing.T) {
	in := make([]uint8, 98)
	for i := range in {
		in[i] = uint8(i & 0x3)
	}
	channel := InterleaveTSBK(in)
	if len(channel) != 98 {
		t.Fatalf("interleaved len = %d, want 98", len(channel))
	}
	back := DeinterleaveTSBK(channel)
	if !bytes.Equal(back, in) {
		t.Errorf("interleave round-trip mismatch")
	}
}

func TestInterleaverPermsAreInverses(t *testing.T) {
	// Verify the two perm tables are proper inverses of each other.
	// Catches typos in the (manually-transcribed) tables.
	for i := 0; i < 98; i++ {
		if tsbkInterleavePerm[tsbkDeinterleavePerm[i]] != i {
			t.Errorf("interleave[deinterleave[%d]] = %d, want %d",
				i, tsbkInterleavePerm[tsbkDeinterleavePerm[i]], i)
		}
		if tsbkDeinterleavePerm[tsbkInterleavePerm[i]] != i {
			t.Errorf("deinterleave[interleave[%d]] = %d, want %d",
				i, tsbkDeinterleavePerm[tsbkInterleavePerm[i]], i)
		}
	}
}

func TestInterleaverIsBijection(t *testing.T) {
	// Every output position must be reached exactly once.
	for _, perm := range []*[98]int{&tsbkInterleavePerm, &tsbkDeinterleavePerm} {
		var seen [98]bool
		for _, j := range perm {
			if j < 0 || j >= 98 {
				t.Fatalf("perm entry out of range: %d", j)
			}
			if seen[j] {
				t.Fatalf("perm entry %d duplicated", j)
			}
			seen[j] = true
		}
	}
}

func TestTSBKChannelRoundTrip(t *testing.T) {
	in := TSBK{
		LB:     true,
		Opcode: OpGroupVoiceChannelGrant,
		MFID:   0x00,
		Payload: [8]byte{
			0x80, 0x10, 0x05, 0x12, 0x34, 0xAB, 0xCD, 0xEF,
		},
	}
	info := AssembleTSBK(in)
	channel := EncodeTSBKChannel(info)
	got, metric, err := DecodeTSBKChannel(channel)
	if err != nil {
		t.Fatalf("DecodeTSBKChannel: %v", err)
	}
	if metric != 0 {
		t.Errorf("clean channel metric = %d, want 0", metric)
	}
	if got != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestTSBKChannelCorrectsErrors(t *testing.T) {
	in := TSBK{
		LB:      true,
		Opcode:  OpRFSSStatusBroadcast,
		Payload: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}
	info := AssembleTSBK(in)
	channel := EncodeTSBKChannel(info)
	// Flip one bit in 4 different channel dibits, well-separated so
	// the deinterleaver scatters them across the trellis stages.
	for _, idx := range []int{5, 25, 51, 77} {
		channel[idx] ^= 0b10
	}
	got, metric, err := DecodeTSBKChannel(channel)
	if err != nil {
		t.Fatalf("DecodeTSBKChannel: %v", err)
	}
	if metric == 0 {
		t.Errorf("metric = 0 after errors, want > 0")
	}
	if got != in {
		t.Errorf("error-correction mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestTSBKChannelDetectsHeavyCorruption(t *testing.T) {
	in := TSBK{
		LB:      true,
		Opcode:  OpRFSSStatusBroadcast,
		Payload: [8]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8},
	}
	info := AssembleTSBK(in)
	channel := EncodeTSBKChannel(info)
	// Heavy noise across the channel: invert every 3rd dibit. This
	// exceeds the trellis correction radius; the CRC will catch the
	// mis-correction.
	for i := 0; i < 98; i += 3 {
		channel[i] = (^channel[i]) & 0x3
	}
	_, _, err := DecodeTSBKChannel(channel)
	if err == nil {
		t.Fatal("heavy corruption should surface an error")
	}
}
